package main

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/network"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func main() {

	runtime.LockOSThread()

	origns, err := netns.Get()
	defer origns.Close()
	if err != nil {
		log.Println(err)
	}

	redBridgeLa := netlink.NewLinkAttrs()
	redBridgeLa.Name = "br-red"
	redBridge := &netlink.Bridge{LinkAttrs: redBridgeLa}

	if err := netlink.LinkAdd(redBridge); err != nil {
		log.Printf("could not add %s: %v\n", redBridge.Name, err)
	}

	if err := netlink.LinkSetUp(redBridge); err != nil {
		log.Printf("could not set %s up: %v\n", redBridge.Name, err)
	}

	blueBridgeLa := netlink.NewLinkAttrs()
	blueBridgeLa.Name = "br-blue"
	blueBridge := &netlink.Bridge{LinkAttrs: blueBridgeLa}

	if err := netlink.LinkAdd(blueBridge); err != nil {
		log.Printf("could not add %s: %v\n", blueBridge.Name, err)
	}

	if err := netlink.LinkSetUp(blueBridge); err != nil {
		log.Printf("could not set %s up: %v\n", blueBridge.Name, err)
	}

	for i := range 2 {
		const subnet int = 0
		name := "red" + strconv.Itoa(i)
		_, err := netns.GetFromName(name)
		if err == nil {
			log.Printf("Namespace '%s' is already present\n", name)
			continue
		}
		ns, err := netns.NewNamed(name)
		if err != nil {
			log.Fatalf("failed to add netns: %v", err)
		}
		netns.Set(origns)

		veth := &netlink.Veth{
			LinkAttrs: netlink.LinkAttrs{
				Name:        name + "-br",
				MasterIndex: redBridge.Index,
			},
			PeerName: name + "-ns",
		}
		if err := netlink.LinkAdd(veth); err != nil {
			log.Fatalf("failed to add veth pair: %v", err)
		}

		if err := netlink.LinkSetUp(veth); err != nil {
			log.Fatalf("failed to set the veth br end up: %v", err)
		}

		vethPeer, err := netlink.LinkByName(name + "-ns")
		if err != nil {
			log.Fatalf("could not get veth peer: %v", err)
		}

		if err := netlink.LinkSetNsFd(vethPeer, int(ns)); err != nil {
			log.Fatalf("failed to move veth peer to its netns: %v", err)
		}

		netns.Set(ns)

		addr, err := netlink.ParseAddr("10.0." + strconv.Itoa(subnet) + "." + strconv.Itoa(i+2) + "/24")
		if err != nil {
			log.Fatalf("failed to parse IP address: %v", err)
		}

		if err := netlink.AddrAdd(vethPeer, addr); err != nil {
			log.Fatalf("failed to add IP addr to the veth peer: %v", err)
		}

		if err := netlink.LinkSetUp(vethPeer); err != nil {
			log.Fatalf("failed to set the veth peer end up: %v", err)
		}

		gw := net.ParseIP("10.0." + strconv.Itoa(subnet) + ".1")
		if gw == nil {
			log.Fatal("Invalid gateway IP")
		}
		defaultRoute := &netlink.Route{
			LinkIndex: vethPeer.Attrs().Index,
			Src:       nil, // default, 0.0.0.0/0
			Gw:        gw,
		}
		if err := netlink.RouteAdd(defaultRoute); err != nil {
			log.Fatalf("Failed to add default route: %v", err)
		}

		netns.Set(origns)

		ns.Close()
	}

	for i := range 3 {
		const subnet int = 1
		name := "blue" + strconv.Itoa(i)
		_, err := netns.GetFromName(name)
		if err == nil {
			log.Printf("Namespace '%s' is already present\n", name)
			continue
		}
		ns, err := netns.NewNamed(name)
		if err != nil {
			log.Fatalf("failed to add netns: %v", err)
		}
		netns.Set(origns)

		veth := &netlink.Veth{
			LinkAttrs: netlink.LinkAttrs{
				Name:        name + "-br",
				MasterIndex: blueBridge.Index,
			},
			PeerName: name + "-ns",
		}
		if err := netlink.LinkAdd(veth); err != nil {
			log.Fatalf("failed to add veth pair: %v", err)
		}

		if err := netlink.LinkSetUp(veth); err != nil {
			log.Fatalf("failed to set the veth br end up: %v", err)
		}

		vethPeer, err := netlink.LinkByName(name + "-ns")
		if err != nil {
			log.Fatalf("could not get veth peer: %v", err)
		}

		if err := netlink.LinkSetNsFd(vethPeer, int(ns)); err != nil {
			log.Fatalf("failed to move veth peer to its netns: %v", err)
		}

		netns.Set(ns)

		addr, err := netlink.ParseAddr("10.0." + strconv.Itoa(subnet) + "." + strconv.Itoa(i+2) + "/24")
		if err != nil {
			log.Fatalf("failed to parse IP address: %v", err)
		}

		if err := netlink.AddrAdd(vethPeer, addr); err != nil {
			log.Fatalf("failed to add IP addr to the veth peer: %v", err)
		}

		if err := netlink.LinkSetUp(vethPeer); err != nil {
			log.Fatalf("failed to set the veth peer end up: %v", err)
		}

		gw := net.ParseIP("10.0." + strconv.Itoa(subnet) + ".1")
		if gw == nil {
			log.Fatal("Invalid gateway IP")
		}
		defaultRoute := &netlink.Route{
			LinkIndex: vethPeer.Attrs().Index,
			Src:       nil, // default, 0.0.0.0/0
			Gw:        gw,
		}
		if err := netlink.RouteAdd(defaultRoute); err != nil {
			log.Fatalf("Failed to add default route: %v", err)
		}

		netns.Set(origns)

		ns.Close()
	}

	routerName := "router"
	_, err = netns.GetFromName(routerName)
	if err != nil {
		ns, err := netns.NewNamed(routerName)
		if err != nil {
			log.Fatalf("failed to add netns: %v", err)
		}
		netns.Set(origns)

		switches := [...]struct {
			bridge *netlink.Bridge
			addr   string
			name   string
		}{struct {
			bridge *netlink.Bridge
			addr   string
			name   string
		}{
			name:   "red",
			bridge: redBridge,
			addr:   "10.0.0.1/24",
		},
			struct {
				bridge *netlink.Bridge
				addr   string
				name   string
			}{
				name:   "blue",
				bridge: blueBridge,
				addr:   "10.0.1.1/24",
			}}

		for _, sw := range switches {
			veth := &netlink.Veth{
				LinkAttrs: netlink.LinkAttrs{
					Name:        routerName + "-" + sw.name + "-br",
					MasterIndex: sw.bridge.Index,
				},
				PeerName: routerName + "-" + sw.name + "-ns",
			}
			if err := netlink.LinkAdd(veth); err != nil {
				log.Fatalf("failed to add veth pair: %v", err)
			}

			if err := netlink.LinkSetUp(veth); err != nil {
				log.Fatalf("failed to set the veth br end up: %v", err)
			}

			vethPeer, err := netlink.LinkByName(routerName + "-" + sw.name + "-ns")
			if err != nil {
				log.Fatalf("could not get veth peer: %v", err)
			}

			if err := netlink.LinkSetNsFd(vethPeer, int(ns)); err != nil {
				log.Fatalf("failed to move veth peer to its netns: %v", err)
			}

			netns.Set(ns)
			addr, err := netlink.ParseAddr(sw.addr)
			if err != nil {
				log.Fatalf("failed to parse IP address: %v", err)
			}

			if err := netlink.AddrAdd(vethPeer, addr); err != nil {
				log.Fatalf("failed to add IP addr to the veth peer: %v", err)
			}

			if err := netlink.LinkSetUp(vethPeer); err != nil {
				log.Fatalf("failed to set the veth peer end up: %v", err)
			}

			if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644); err != nil {
				log.Fatalf("failed to enable ip forwarding: %v", err)
			}

			netns.Set(origns)
		}

		ns.Close()
	} else {
		log.Printf("Namespace '%s' is already present\n", routerName)
	}

	// cmd := exec.Command("sleep", "10000")
	// cmd.Start()

	// Do something with the network namespace
	ifaces, _ := net.Interfaces()
	log.Printf("Interfaces: %v\n", ifaces)

	// Switch back to the original namespace
	netns.Set(origns)

	runtime.UnlockOSThread()

	address := "localhost:8080"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	var sampleNetworks = []*pb.Network{
		&pb.Network{
			Id:             id.NextId("network"),
			InternetAccess: false,
			CreatedAt:      timestamppb.New(time.Now().Add(-time.Hour)),
		},
		&pb.Network{
			Id:             id.NextId("network"),
			InternetAccess: true,
			CreatedAt:      timestamppb.New(time.Now().Add(-time.Minute)),
		},
		&pb.Network{
			Id:             id.NextId("network"),
			InternetAccess: true,
			CreatedAt:      timestamppb.New(time.Now().Add(-time.Minute * 30)),
		},
	}
	networkRepository := network.NewMemoryNetworkRepository(sampleNetworks)

	var sampleSubnetworks = []*pb.Subnetwork{
		&pb.Subnetwork{
			Id:           id.NextId("subnetwork"),
			NetworkId:    sampleNetworks[0].Id,
			Address:      binary.BigEndian.Uint32([]byte{10, 0, 0, 0}),
			PrefixLength: 24,
			CreatedAt:    timestamppb.New(time.Now().Add(-time.Hour)),
		},
		&pb.Subnetwork{
			Id:           id.NextId("subnetwork"),
			NetworkId:    sampleNetworks[0].Id,
			Address:      binary.BigEndian.Uint32([]byte{10, 0, 1, 0}),
			PrefixLength: 24,
			CreatedAt:    timestamppb.New(time.Now().Add(-time.Minute)),
		},
		&pb.Subnetwork{
			Id:           id.NextId("subnetwork"),
			NetworkId:    sampleNetworks[2].Id,
			Address:      binary.BigEndian.Uint32([]byte{192, 168, 0, 64}),
			PrefixLength: 26,
			CreatedAt:    timestamppb.New(time.Now().Add(-time.Minute * 29)),
		},
	}
	subnetworkRepository := subnetwork.NewMemorySubnetworkRepository(sampleSubnetworks)

	pb.RegisterNetworkServiceServer(grpcServer, network.NewNetworkService(networkRepository, subnetworkRepository))
	pb.RegisterSubnetworkServiceServer(grpcServer, subnetwork.NewSubnetworkService(subnetworkRepository, networkRepository))

	log.Printf("Starting server on %s\n", address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
