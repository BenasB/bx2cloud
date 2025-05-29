package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"text/tabwriter"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/cli/inputs"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v3"
)

func newNetworkWriter() *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "id\tinternetAccess\n")
	return w
}

func printNetwork(w *tabwriter.Writer, network *pb.Network) {
	fmt.Fprintf(w, "%d\t%t\n", network.Id, network.InternetAccess)
}

func newSubnetworkWriter() *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "id\tnetwork_id\tcidr\n")
	return w
}

func printSubnetwork(w *tabwriter.Writer, subnetwork *pb.Subnetwork) {
	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(subnetwork.Address>>24),
		byte(subnetwork.Address>>16),
		byte(subnetwork.Address>>8),
		byte(subnetwork.Address),
		subnetwork.PrefixLength)

	fmt.Fprintf(w, "%d\t%d\t%s\n", subnetwork.Id, subnetwork.NetworkId, cidr)
}

func networkList(client pb.NetworkServiceClient) error {
	stream, err := client.List(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	w := newNetworkWriter()
	defer w.Flush()
	for {
		network, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		printNetwork(w, network)
	}

	return nil
}

func networkGet(client pb.NetworkServiceClient, id uint32) error {
	network, err := client.Get(context.Background(), &pb.NetworkIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	w := newNetworkWriter()
	defer w.Flush()
	printNetwork(w, network)

	return nil
}

func networkDelete(client pb.NetworkServiceClient, id uint32) error {
	_, err := client.Delete(context.Background(), &pb.NetworkIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Successfully deleted %d\n", id)

	return nil
}

func networkCreate(client pb.NetworkServiceClient, yamlBytes []byte) error {
	input := &inputs.NetworkCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	req := &pb.NetworkCreationRequest{
		InternetAccess: input.InternetAccess,
	}

	resp, err := client.Create(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created %d\n", resp.Id)

	return nil
}

func networkUpdate(client pb.NetworkServiceClient, id uint32, yamlBytes []byte) error {
	input := &inputs.NetworkCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	req := &pb.NetworkUpdateRequest{
		Identification: &pb.NetworkIdentificationRequest{
			Id: id,
		},
		Update: &pb.NetworkCreationRequest{
			InternetAccess: input.InternetAccess,
		},
	}

	resp, err := client.Update(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully updated %d\n", resp.Id)

	return nil
}

func subnetworkList(client pb.SubnetworkServiceClient) error {
	stream, err := client.List(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	w := newSubnetworkWriter()
	defer w.Flush()
	for {
		subnetwork, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		printSubnetwork(w, subnetwork)
	}

	return nil
}

func subnetworkGet(client pb.SubnetworkServiceClient, id uint32) error {
	subnetwork, err := client.Get(context.Background(), &pb.SubnetworkIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	w := newSubnetworkWriter()
	defer w.Flush()
	printSubnetwork(w, subnetwork)

	return nil
}

func subnetworkDelete(client pb.SubnetworkServiceClient, id uint32) error {
	_, err := client.Delete(context.Background(), &pb.SubnetworkIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Successfully deleted %d\n", id)

	return nil
}

func subnetworkCreate(client pb.SubnetworkServiceClient, yamlBytes []byte) error {
	input := &inputs.SubnetworkCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	_, ipNet, err := net.ParseCIDR(input.Cidr)
	if err != nil {
		return fmt.Errorf("Could not parse CIDR: %v", err)
	}

	ip := ipNet.IP.To4()
	if ip == nil {
		return fmt.Errorf("Could not convert the ip to an IPv4 ip")
	}
	address := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
	prefixLength, _ := ipNet.Mask.Size()

	req := &pb.SubnetworkCreationRequest{
		NetworkId:    input.NetworkId,
		Address:      address,
		PrefixLength: uint32(prefixLength),
	}

	resp, err := client.Create(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created %d\n", resp.Id)

	return nil
}

func subnetworkUpdate(client pb.SubnetworkServiceClient, id uint32, yamlBytes []byte) error {
	input := &inputs.SubnetworkCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	_, ipNet, err := net.ParseCIDR(input.Cidr)
	if err != nil {
		return fmt.Errorf("Could not parse CIDR: %v", err)
	}

	ip := ipNet.IP.To4()
	if ip == nil {
		return fmt.Errorf("Could not convert the ip to an IPv4 ip")
	}
	address := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
	prefixLength, _ := ipNet.Mask.Size()

	req := &pb.SubnetworkUpdateRequest{
		Identification: &pb.SubnetworkIdentificationRequest{
			Id: id,
		},
		Update: &pb.SubnetworkCreationRequest{
			Address:      address,
			PrefixLength: uint32(prefixLength),
		},
	}

	resp, err := client.Update(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully updated %d\n", resp.Id)

	return nil
}
