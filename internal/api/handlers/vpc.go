package handlers

import (
	"context"
	"fmt"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type VpcService struct {
	pb.UnimplementedVpcServiceServer
	vpcs []*pb.Vpc
}

func NewVpcService(vpcs []*pb.Vpc) *VpcService {
	serviceVpcs := make([]*pb.Vpc, len(vpcs))
	copy(serviceVpcs, vpcs)
	return &VpcService{
		vpcs: serviceVpcs,
	}
}

func (s *VpcService) Get(ctx context.Context, req *pb.VpcIdentificationRequest) (*pb.Vpc, error) {
	for _, vpc := range s.vpcs {
		match := false
		switch x := req.Identification.(type) {
		case *pb.VpcIdentificationRequest_Id:
			match = vpc.Id == x.Id

		case *pb.VpcIdentificationRequest_Name:
			match = vpc.Name == x.Name
		}

		if match {
			return vpc, nil
		}
	}

	return nil, fmt.Errorf("could not find VPC")
}

func (s *VpcService) Delete(ctx context.Context, req *pb.VpcIdentificationRequest) (*emptypb.Empty, error) {
	for i, vpc := range s.vpcs {
		match := false
		switch x := req.Identification.(type) {
		case *pb.VpcIdentificationRequest_Id:
			match = vpc.Id == x.Id

		case *pb.VpcIdentificationRequest_Name:
			match = vpc.Name == x.Name
		}

		if match {
			s.vpcs = append(s.vpcs[:i], s.vpcs[i+1:]...)
			return &emptypb.Empty{}, nil
		}
	}

	return nil, fmt.Errorf("could not find VPC")
}

func (s *VpcService) Create(ctx context.Context, req *pb.VpcCreationRequest) (*pb.Vpc, error) {
	newVpc := &pb.Vpc{
		Id:        uuid.NewString(),
		Name:      req.Name,
		Cidr:      req.Cidr,
		CreatedAt: timestamppb.New(time.Now()),
	}

	s.vpcs = append(s.vpcs, newVpc)

	return newVpc, nil
}
