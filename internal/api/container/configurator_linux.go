package container

import (
	"fmt"
	"log"
	"net"
	"runtime"
	"strconv"

	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type configurator interface {
	configure(model *shared.ContainerModel, subnetworkModel *shared.SubnetworkModel) error
	unconfigure(model *shared.ContainerModel, subnetworkModel *shared.SubnetworkModel) error
}

var _ configurator = &namespaceConfigurator{}

type namespaceConfigurator struct {
	getNetworkNamespaceName func(uint32) string
	getBridgeName           func(uint32) string
}

func NewNamespaceConfigurator(getNetworkNamespaceName func(uint32) string, getBridgeName func(uint32) string) *namespaceConfigurator {
	return &namespaceConfigurator{
		getNetworkNamespaceName: getNetworkNamespaceName,
		getBridgeName:           getBridgeName,
	}
}

func (n *namespaceConfigurator) configure(model *shared.ContainerModel, subnetworkModel *shared.SubnetworkModel) error {
	networkNsName := n.getNetworkNamespaceName(subnetworkModel.NetworkId)
	networkNs, err := netns.GetFromName(networkNsName)
	defer networkNs.Close()
	if err != nil {
		return fmt.Errorf("failed to retrieve the network's namespace: %w", err)
	}

	state, err := model.OCIState()
	if err != nil {
		return fmt.Errorf("failed to retrieve the container's current state: %w", err)
	}

	containerNsPath := (&configs.Namespace{Type: configs.NEWNET}).GetPath(state.Pid)
	containerNs, err := netns.GetFromPath(containerNsPath)
	defer containerNs.Close()
	if err != nil {
		return fmt.Errorf("failed to retrieve the network namespace of the container from the file path: %w", err)
	}

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

	if err := netns.Set(networkNs); err != nil {
		return fmt.Errorf("failed to switch to the network's namespace: %w", err)
	}

	bridgeName := n.getBridgeName(subnetworkModel.Id)
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("failed to retrieve the subnetwork's bridge in the network's namespace: %w", err)
	}

	networkVethName := n.getNetworkVethName(model)
	containerVethName := n.getContainerVethName(model)
	networkVeth, err := netlink.LinkByName(networkVethName)
	if err != nil {
		la := netlink.NewLinkAttrs()
		la.Name = networkVethName
		la.MasterIndex = bridge.Attrs().Index
		containerVethCreation := &netlink.Veth{
			LinkAttrs:     la,
			PeerName:      containerVethName,
			PeerNamespace: netlink.NsFd(containerNs),
		}

		if err := netlink.LinkAdd(containerVethCreation); err != nil {
			return fmt.Errorf("failed to add a veth pair when connecting the network's namespace to the container's namespace: %w", err)
		}

		networkVeth = containerVethCreation
	}

	if networkVeth.Attrs().OperState != netlink.OperUp {
		if err := netlink.LinkSetUp(networkVeth); err != nil {
			return fmt.Errorf("failed to set the network's namespace veth end up: %w", err)
		}
	}

	// TODO: Go into container namespace, add ip to interface, interface up, add default gateway route
	if err := netns.Set(containerNs); err != nil {
		return fmt.Errorf("failed to switch to the network namespace of the container: %w", err)
	}

	containerVeth, err := netlink.LinkByName(containerVethName)
	if err != nil {
		return fmt.Errorf("failed to get the container's namespace veth end: %w", err)
	}

	containerVethAddrs, err := netlink.AddrList(containerVeth, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve IP addresses of the container's namespace veth end: %w", err)
	}

	// TODO: IPAM
	id, err := strconv.ParseUint(model.ID(), 10, 32)
	if err != nil {
		return fmt.Errorf("failed to convert the container's ID to an integer: %w", err)
	}
	containerVethIp := subnetworkModel.Address + 1 + uint32(id)
	containerVethAddr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   net.IPv4(byte(containerVethIp>>24), byte(containerVethIp>>16), byte(containerVethIp>>8), byte(containerVethIp)),
			Mask: net.CIDRMask(int(subnetworkModel.PrefixLength), 32),
		},
	}

	var containerVethIpExists = false
	for _, addr := range containerVethAddrs {
		if containerVethAddr.Equal(addr) {
			containerVethIpExists = true
			continue
		}

		if err := netlink.AddrDel(containerVeth, &addr); err != nil {
			return fmt.Errorf("failed to remove an unexpected IP address from the container's namespace veth end: %w", err)
		}
	}

	if !containerVethIpExists {
		if err := netlink.AddrAdd(containerVeth, containerVethAddr); err != nil {
			return fmt.Errorf("failed to add an IP address to the container's namespace veth end: %w", err)
		}
	}

	if containerVeth.Attrs().OperState != netlink.OperUp {
		if err := netlink.LinkSetUp(containerVeth); err != nil {
			return fmt.Errorf("failed to set the container's namespace veth end up: %w", err)
		}
	}

	gwIp := subnetworkModel.Address + 1 // TODO: IPAM
	defaultRoute := &netlink.Route{
		LinkIndex: containerVeth.Attrs().Index,
		Dst: &net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.CIDRMask(0, 32),
		}, // default, 0.0.0.0/0
		Gw: net.IPv4(byte(gwIp>>24), byte(gwIp>>16), byte(gwIp>>8), byte(gwIp)),
	}

	routes, err := netlink.RouteList(containerVeth, netlink.FAMILY_V4)
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

	if err := netns.Set(origNs); err != nil {
		return fmt.Errorf("failed to switch to the original network namespace: %w", err)
	}

	log.Printf("Successfully configured container with the id %s", model.ID())

	return nil
}

func (n *namespaceConfigurator) unconfigure(model *shared.ContainerModel, subnetworkModel *shared.SubnetworkModel) error {
	panic("unimplemented")

	log.Printf("Successfully unconfigured container with the id %s", model.ID())

	return nil
}

func (n *namespaceConfigurator) getNetworkVethName(model *shared.ContainerModel) string {
	return fmt.Sprintf("bx2-c-%s", model.ID())
}

func (n *namespaceConfigurator) getContainerVethName(model *shared.ContainerModel) string {
	return fmt.Sprintf("bx2-c-%s-ns", model.ID())
}
