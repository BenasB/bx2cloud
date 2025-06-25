package network

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var _ configurator = &namespaceConfigurator{}

type namespaceConfigurator struct {
	primaryInterface netlink.Link
	ipt              *iptables.IPTables
}

func NewNamespaceConfigurator() (*namespaceConfigurator, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes when locating the primary interface: %w", err)
	}

	var primaryInterface netlink.Link
	for _, route := range routes {
		if route.Dst.IP.Equal(net.IPv4zero) {
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to get the primary interface: %w", err)
			}
			primaryInterface = link
			break
		}
	}

	if primaryInterface == nil {
		return nil, fmt.Errorf("failed to find the primary interface")
	}

	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644); err != nil {
		return nil, fmt.Errorf("failed to enable ip forwarding in the root namespace: %w", err)
	}

	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return nil, fmt.Errorf("failed to create iptables instance: %w", err)
	}

	err = ipt.AppendUnique("filter", "FORWARD",
		"-i", "bx2-r-+",
		"-o", "bx2-r-+",
		"-j", "DROP",
	)

	if err != nil {
		return nil, fmt.Errorf("Failed to add DROP rule for traffic between bx2cloud networks: %w", err)
	}

	return &namespaceConfigurator{
		primaryInterface: primaryInterface,
		ipt:              ipt,
	}, nil
}

func (n *namespaceConfigurator) Configure(model *shared.NetworkModel) error {
	nsName := n.GetNetworkNamespaceName(model.Id)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origNs, err := netns.Get()
	defer origNs.Close()
	if err != nil {
		return fmt.Errorf("failed to retrieve the original network namespace: %w", err)
	}
	defer func() {
		if err := netns.Set(origNs); err != nil {
			panic("failed to move back to the original network namespace, panicking to not change unexpected state")
		}
	}()

	ns, err := netns.GetFromName(nsName)
	defer ns.Close()
	if err != nil {
		ns, err = netns.NewNamed(nsName)
		if err != nil {
			return fmt.Errorf("failed to create a network namespace for the network: %w", err)
		}

		if err := netns.Set(origNs); err != nil {
			return fmt.Errorf("failed to switch back to the root network namespace: %w", err)
		}
	}

	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to switch to the network's namespace: %w", err)
	}

	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644); err != nil {
		return fmt.Errorf("failed to enable ip forwarding: %w", err)
	}

	if err := netns.Set(origNs); err != nil {
		return fmt.Errorf("failed to switch back to the root network namespace: %w", err)
	}

	if model.InternetAccess {
		if err := n.configureInternetAccess(model, origNs, ns); err != nil {
			return err
		}
	} else {
		if err := n.unconfigureInternetAccess(model, origNs, ns); err != nil {
			return err
		}
	}

	log.Printf("Successfully configured network with the id %d", model.Id)

	return nil
}

func (n *namespaceConfigurator) Unconfigure(model *shared.NetworkModel) error {
	nsName := n.GetNetworkNamespaceName(model.Id)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origNs, err := netns.Get()
	defer origNs.Close()
	if err != nil {
		return fmt.Errorf("failed to retrieve the original network namespace: %w", err)
	}
	defer func() {
		if err := netns.Set(origNs); err != nil {
			panic("failed to move back to the original network namespace, panicking to not change unexpected state")
		}
	}()

	ns, nsErr := netns.GetFromName(nsName)
	defer ns.Close()

	if err := n.unconfigureInternetAccess(model, origNs, ns); err != nil {
		return err
	}

	if nsErr == nil {
		if err := netns.DeleteNamed(nsName); err != nil {
			return fmt.Errorf("failed to delete the network namespace: %w", err)
		}
	}

	log.Printf("Successfully unconfigured network with the id %d", model.Id)

	return nil
}

func (n *namespaceConfigurator) configureInternetAccess(model *shared.NetworkModel, origNs netns.NsHandle, ns netns.NsHandle) error {
	rootVethName := n.getRootVethName(model)
	nsVethName := n.getNsVethName(model)

	rootVeth, err := netlink.LinkByName(rootVethName)
	if err != nil {
		la := netlink.NewLinkAttrs()
		la.Name = rootVethName
		rootVethCreation := &netlink.Veth{
			LinkAttrs:     la,
			PeerName:      nsVethName,
			PeerNamespace: netlink.NsFd(ns),
		}

		if err := netlink.LinkAdd(rootVethCreation); err != nil {
			return fmt.Errorf("failed to add a veth pair when connecting the network's namespace to the root namespace: %w", err)
		}

		rootVeth = rootVethCreation
	}

	rootVethAddrs, err := netlink.AddrList(rootVeth, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve IP addresses of the root namespace veth end: %w", err)
	}

	rootVethAddr := n.getRootVethAddr(model)

	var rootVethIpExists = false
	for _, addr := range rootVethAddrs {
		if rootVethAddr.Equal(addr) {
			rootVethIpExists = true
			continue
		}

		if err := netlink.AddrDel(rootVeth, &addr); err != nil {
			return fmt.Errorf("failed to remove an unexpected IP address from the root namespace veth end: %w", err)
		}
	}

	if !rootVethIpExists {
		if err := netlink.AddrAdd(rootVeth, rootVethAddr); err != nil {
			return fmt.Errorf("failed to add an IP address to the root namespace veth end: %w", err)
		}
	}

	if rootVeth.Attrs().OperState != netlink.OperUp {
		if err := netlink.LinkSetUp(rootVeth); err != nil {
			return fmt.Errorf("failed to set the root namespace veth end up: %w", err)
		}
	}

	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to switch to the network's namespace: %w", err)
	}

	nsVeth, err := netlink.LinkByName(nsVethName)
	if err != nil {
		return fmt.Errorf("failed to get the network's namespace veth end: %w", err)
	}

	nsVethAddrs, err := netlink.AddrList(nsVeth, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve IP addresses of the network's namespace veth end: %w", err)
	}

	nsVethAddr := n.getNsVethAddr(model)

	var nsVethIpExists = false
	for _, addr := range nsVethAddrs {
		if nsVethAddr.Equal(addr) {
			nsVethIpExists = true
			continue
		}

		if err := netlink.AddrDel(nsVeth, &addr); err != nil {
			return fmt.Errorf("failed to remove an unexpected IP address from the network's namespace veth end: %w", err)
		}
	}

	if !nsVethIpExists {
		if err := netlink.AddrAdd(nsVeth, nsVethAddr); err != nil {
			return fmt.Errorf("failed to add an IP address to the network's namespace veth end: %w", err)
		}
	}

	if nsVeth.Attrs().OperState != netlink.OperUp {
		if err := netlink.LinkSetUp(nsVeth); err != nil {
			return fmt.Errorf("failed to set the network's namespace veth end up: %w", err)
		}
	}

	defaultRoute := &netlink.Route{
		LinkIndex: nsVeth.Attrs().Index,
		Dst: &net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.CIDRMask(0, 32),
		}, // default, 0.0.0.0/0
		Gw: rootVethAddr.IP,
	}

	routes, err := netlink.RouteList(nsVeth, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve routes of the network's namespace: %w", err)
	}

	var defaultRouteExists = false
	for _, route := range routes {
		if defaultRoute.LinkIndex == route.LinkIndex &&
			defaultRoute.Dst.IP.Equal(route.Dst.IP) &&
			defaultRoute.Gw.Equal(route.Gw) {
			defaultRouteExists = true
			continue
		}
	}

	if !defaultRouteExists {
		if err := netlink.RouteAdd(defaultRoute); err != nil {
			return fmt.Errorf("failed to add the default route: %w", err)
		}
	}

	err = n.ipt.AppendUnique("nat", "POSTROUTING",
		"-o", nsVeth.Attrs().Name,
		"-j", "MASQUERADE",
	)

	if err != nil {
		return fmt.Errorf("Failed to add SNAT rule for subnetwork translation: %w", err)
	}

	if err := netns.Set(origNs); err != nil {
		return fmt.Errorf("failed to switch back to the root network namespace: %w", err)
	}

	err = n.ipt.AppendUnique("nat", "POSTROUTING",
		"-s", nsVethAddr.IPNet.String(),
		"-o", n.primaryInterface.Attrs().Name,
		"-j", "MASQUERADE",
	)

	if err != nil {
		return fmt.Errorf("Failed to add SNAT rule on the primary interface: %w", err)
	}

	return nil
}

func (n *namespaceConfigurator) unconfigureInternetAccess(model *shared.NetworkModel, origNs netns.NsHandle, ns netns.NsHandle) error {
	if ns.IsOpen() {
		if err := netns.Set(ns); err != nil {
			return fmt.Errorf("failed to switch to the network's namespace: %w", err)
		}

		nsVeth, err := netlink.LinkByName(n.getNsVethName(model))
		if err == nil {
			const networkIpStart uint32 = 0b_11000000_10100111_00000000_00000000
			networkIp := networkIpStart + model.Id<<2
			rootVethIp := networkIp + 1

			rootVethAddr := &netlink.Addr{
				IPNet: &net.IPNet{
					IP:   net.IPv4(byte(rootVethIp>>24), byte(rootVethIp>>16), byte(rootVethIp>>8), byte(rootVethIp)),
					Mask: net.CIDRMask(30, 32),
				},
			}

			defaultRoute := &netlink.Route{
				LinkIndex: nsVeth.Attrs().Index,
				Dst: &net.IPNet{
					IP:   net.IPv4zero,
					Mask: net.CIDRMask(0, 32),
				}, // default, 0.0.0.0/0
				Gw: rootVethAddr.IP,
			}

			routes, err := netlink.RouteList(nsVeth, netlink.FAMILY_V4)
			if err != nil {
				return fmt.Errorf("failed to retrieve routes of the network's namespace: %w", err)
			}

			for _, route := range routes {
				if defaultRoute.LinkIndex == route.LinkIndex &&
					defaultRoute.Dst.IP.Equal(route.Dst.IP) &&
					defaultRoute.Gw.Equal(route.Gw) {
					if netlink.RouteDel(&route); err != nil {
						return fmt.Errorf("failed to remove the default route: %w", err)
					}
					break
				}
			}

			err = n.ipt.DeleteIfExists("nat", "POSTROUTING",
				"-o", nsVeth.Attrs().Name,
				"-j", "MASQUERADE",
			)

			if err != nil {
				return fmt.Errorf("Failed to remove SNAT rule for subnetwork translation: %w", err)
			}
		}

		if err := netns.Set(origNs); err != nil {
			return fmt.Errorf("failed to switch back to the root network namespace: %w", err)
		}
	}

	rootVeth, err := netlink.LinkByName(n.getRootVethName(model))
	if err == nil {
		if netlink.LinkDel(rootVeth); err != nil {
			return fmt.Errorf("failed to remove the veth pair: %w", err)
		}
	}

	nsVethAddr := n.getNsVethAddr(model)
	err = n.ipt.DeleteIfExists("nat", "POSTROUTING",
		"-s", nsVethAddr.IPNet.String(),
		"-o", n.primaryInterface.Attrs().Name,
		"-j", "MASQUERADE",
	)

	if err != nil {
		return fmt.Errorf("Failed to remove SNAT rule on the primary interface: %w", err)
	}

	return nil
}

func (n *namespaceConfigurator) GetNetworkNamespaceName(id uint32) string {
	return fmt.Sprintf("bx2cloud-router-%d", id)
}

func (n *namespaceConfigurator) getRootVethName(model *shared.NetworkModel) string {
	return fmt.Sprintf("bx2-r-%d", model.Id)
}

func (n *namespaceConfigurator) getNsVethName(model *shared.NetworkModel) string {
	return fmt.Sprintf("bx2-r-%d-ns", model.Id)
}

func (n *namespaceConfigurator) getRootVethAddr(model *shared.NetworkModel) *netlink.Addr {
	const networkIpStart uint32 = 0b_11000000_10100111_00000000_00000000
	networkIp := networkIpStart + model.Id<<2
	vethIp := networkIp + 1

	return &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   net.IPv4(byte(vethIp>>24), byte(vethIp>>16), byte(vethIp>>8), byte(vethIp)),
			Mask: net.CIDRMask(30, 32),
		},
	}
}

func (n *namespaceConfigurator) getNsVethAddr(model *shared.NetworkModel) *netlink.Addr {
	const networkIpStart uint32 = 0b_11000000_10100111_00000000_00000000
	networkIp := networkIpStart + model.Id<<2
	vethIp := networkIp + 2

	return &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   net.IPv4(byte(vethIp>>24), byte(vethIp>>16), byte(vethIp>>8), byte(vethIp)),
			Mask: net.CIDRMask(30, 32),
		},
	}
}
