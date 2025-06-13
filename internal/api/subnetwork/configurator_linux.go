package subnetwork

import (
	"fmt"
	"log"
	"runtime"

	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var _ configurator = &bridgeConfigurator{}

type bridgeConfigurator struct {
	getNetworkNamespaceName func(uint32) string
	ipamRepository          shared.IpamRepository
}

func NewBridgeConfigurator(getNetworkNamespaceName func(uint32) string, ipamRepository shared.IpamRepository) *bridgeConfigurator {
	return &bridgeConfigurator{
		getNetworkNamespaceName: getNetworkNamespaceName,
		ipamRepository:          ipamRepository,
	}
}

func (b *bridgeConfigurator) configure(model *shared.SubnetworkModel) error {
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

	netNsName := b.getNetworkNamespaceName(model.NetworkId)
	netNs, err := netns.GetFromName(netNsName)
	defer netNs.Close()
	if err != nil {
		return fmt.Errorf("failed to get the network namespace for the network: %w", err)
	}

	if err := netns.Set(netNs); err != nil {
		return fmt.Errorf("failed to switch to the network's namespace: %w", err)
	}

	bridgeName := b.GetBridgeName(model.Id)
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		la := netlink.NewLinkAttrs()
		la.Name = bridgeName
		bridgeCreation := &netlink.Bridge{
			LinkAttrs: la,
		}

		if err := netlink.LinkAdd(bridgeCreation); err != nil {
			return fmt.Errorf("failed to add a bridge interface for the subnetwork: %w", err)
		}

		bridge = bridgeCreation
	}

	bridgeAddrs, err := netlink.AddrList(bridge, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve IP addresses of the bridge: %w", err)
	}

	bridgeAddr := &netlink.Addr{
		IPNet: b.ipamRepository.GetSubnetworkGateway(model),
	}

	var expectedIpExists = false
	for _, addr := range bridgeAddrs {
		if bridgeAddr.Equal(addr) {
			expectedIpExists = true
			continue
		}

		if err := netlink.AddrDel(bridge, &addr); err != nil {
			return fmt.Errorf("failed to remove an unexpected IP address from the bridge: %w", err)
		}
	}

	if !expectedIpExists {
		if err := netlink.AddrAdd(bridge, bridgeAddr); err != nil {
			return fmt.Errorf("failed to add an IP address to the bridge: %w", err)
		}
	}

	if bridge.Attrs().OperState != netlink.OperUp {
		if err := netlink.LinkSetUp(bridge); err != nil {
			return fmt.Errorf("failed to set the bridge interface up: %w", err)
		}
	}

	if err := netns.Set(origNs); err != nil {
		return fmt.Errorf("failed to switch back to the root network namespace: %w", err)
	}

	log.Printf("Successfully configured subnetwork with the id %d", model.Id)

	return nil
}

func (b *bridgeConfigurator) unconfigure(model *shared.SubnetworkModel) error {
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

	netNsName := b.getNetworkNamespaceName(model.NetworkId)
	netNs, err := netns.GetFromName(netNsName)
	defer netNs.Close()
	if err != nil {
		return fmt.Errorf("failed to get the network namespace for the network: %w", err)
	}

	if err := netns.Set(netNs); err != nil {
		return fmt.Errorf("failed to switch to the network's namespace: %w", err)
	}

	bridge, err := netlink.LinkByName(b.GetBridgeName(model.Id))
	if err == nil {
		if netlink.LinkDel(bridge); err != nil {
			return fmt.Errorf("failed to remove the bridge interface: %w", err)
		}
	}

	if err := netns.Set(origNs); err != nil {
		return fmt.Errorf("failed to switch to the root network namespace: %w", err)
	}

	log.Printf("Successfully unconfigured subnetwork with the id %d", model.Id)

	return nil
}

func (b *bridgeConfigurator) GetBridgeName(id uint32) string {
	return fmt.Sprintf("bx2-br-%d", id)
}
