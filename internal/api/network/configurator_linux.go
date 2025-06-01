package network

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var _ configurator = &namespaceConfigurator{}

type namespaceConfigurator struct{}

func NewNamespaceConfigurator() configurator {
	return &namespaceConfigurator{}
}

func (n *namespaceConfigurator) configure(model *shared.NetworkModel) error {
	nsName := fmt.Sprintf("bx2cloud-router-%d", model.Id)
	origNs, err := netns.Get()
	defer origNs.Close()

	if err != nil {
		return fmt.Errorf("failed to retrieve the original network namespace: %w", err)
	}

	ns, err := netns.GetFromName(nsName)
	defer ns.Close()
	if err != nil {
		runtime.LockOSThread()
		ns, err = netns.NewNamed(nsName)
		if err != nil {
			return fmt.Errorf("failed to create a network namespace for the network: %w", err)
		}

		netns.Set(origNs)
		runtime.UnlockOSThread()
	}

	runtime.LockOSThread()
	netns.Set(ns)

	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644); err != nil {
		return fmt.Errorf("failed to enable ip forwarding: %w", err)
	}

	netns.Set(origNs)
	runtime.UnlockOSThread()

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

func (n *namespaceConfigurator) unconfigure(model *shared.NetworkModel) error {
	nsName := fmt.Sprintf("bx2cloud-router-%d", model.Id)

	origNs, err := netns.Get()
	defer origNs.Close()
	if err != nil {
		return fmt.Errorf("failed to retrieve the original network namespace: %w", err)
	}

	ns, err := netns.GetFromName(nsName)
	defer ns.Close()
	if err != nil {
		return fmt.Errorf("failed to get the network namespace for the network: %w", err)
	}

	if err := n.unconfigureInternetAccess(model, origNs, ns); err != nil {
		return err
	}

	if netns.DeleteNamed(nsName) != nil {
		return fmt.Errorf("failed to delete the network namespace: %w", err)
	}

	log.Printf("Successfully unconfigured network with the id %d", model.Id)

	return nil
}

func (n *namespaceConfigurator) configureInternetAccess(model *shared.NetworkModel, origNs netns.NsHandle, ns netns.NsHandle) error {
	rootVethName := fmt.Sprintf("bx2-r-%d", model.Id)
	nsVethName := fmt.Sprintf("%s-ns", rootVethName)
	const networkIpStart uint32 = 0b_11000000_10100111_00000000_00000000
	networkIp := networkIpStart + model.Id<<2
	rootVethIp := networkIp + 1
	nsVethIp := networkIp + 2

	rootVethAddr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   net.IPv4(byte(rootVethIp>>24), byte(rootVethIp>>16), byte(rootVethIp>>8), byte(rootVethIp)),
			Mask: net.CIDRMask(30, 32),
		},
	}

	nsVethAddr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   net.IPv4(byte(nsVethIp>>24), byte(nsVethIp>>16), byte(nsVethIp>>8), byte(nsVethIp)),
			Mask: net.CIDRMask(30, 32),
		},
	}

	la := netlink.NewLinkAttrs()
	la.Name = rootVethName
	rootVethCreation := &netlink.Veth{
		LinkAttrs:     la,
		PeerName:      nsVethName,
		PeerNamespace: netlink.NsFd(ns),
	}

	rootVeth, err := netlink.LinkByName(rootVethName)
	if err != nil {
		if err := netlink.LinkAdd(rootVethCreation); err != nil {
			return fmt.Errorf("failed to add a veth pair when connecting the network's namespace to the root namespace: %w", err)
		}
		rootVeth = rootVethCreation
	}

	rootVethAddrs, err := netlink.AddrList(rootVeth, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve IP addresses of the root namespace veth end: %w", err)
	}

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

	runtime.LockOSThread()
	netns.Set(ns)

	nsVeth, err := netlink.LinkByName(nsVethName)
	if err != nil {
		return fmt.Errorf("failed to get the network's namespace veth end: %w", err)
	}

	nsVethAddrs, err := netlink.AddrList(nsVeth, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve IP addresses of the network's namespace veth end: %w", err)
	}

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

	netns.Set(origNs)
	runtime.UnlockOSThread()

	// TODO: Masquerade

	return nil
}

func (n *namespaceConfigurator) unconfigureInternetAccess(model *shared.NetworkModel, origNs netns.NsHandle, ns netns.NsHandle) error {
	rootVethName := fmt.Sprintf("bx2-r-%d", model.Id)
	nsVethName := fmt.Sprintf("%s-ns", rootVethName)

	rootVeth, err := netlink.LinkByName(rootVethName)
	if err == nil {
		if netlink.LinkDel(rootVeth); err != nil {
			return fmt.Errorf("failed to remove the veth pair: %w", err)
		}
	}

	runtime.LockOSThread()
	netns.Set(ns)

	nsVeth, err := netlink.LinkByName(nsVethName)
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
	}

	netns.Set(origNs)
	runtime.UnlockOSThread()

	// TODO: Masquerade

	return nil
}
