package handlers

import (
	"testing"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testVpcs = []*pb.Vpc{
	&pb.Vpc{
		Id:        "abc-f12",
		Name:      "first-vpc",
		Cidr:      "10.0.1.0/24",
		CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&pb.Vpc{
		Id:        "def-h41x21",
		Name:      "second-vpc",
		Cidr:      "10.0.2.0/24",
		CreatedAt: timestamppb.New(time.Now().Add(-time.Minute)),
	},
}

var identifyById = func(v *pb.Vpc) *pb.VpcIdentificationRequest {
	return &pb.VpcIdentificationRequest{
		Identification: &pb.VpcIdentificationRequest_Id{
			Id: v.Id,
		},
	}
}

var identifyByName = func(v *pb.Vpc) *pb.VpcIdentificationRequest {
	return &pb.VpcIdentificationRequest{
		Identification: &pb.VpcIdentificationRequest_Name{
			Name: v.Name,
		},
	}
}

var identificationFunctions = []func(v *pb.Vpc) *pb.VpcIdentificationRequest{identifyById, identifyByName}

type mockStream[T any] struct {
	grpc.ServerStream
	SentItems []T
}

func (s *mockStream[T]) Send(item T) error {
	s.SentItems = append(s.SentItems, item)
	return nil
}

func TestCreate(t *testing.T) {
	service := NewVpcService(make([]*pb.Vpc, 0))
	req := &pb.VpcCreationRequest{
		Name: "my-vpc",
		Cidr: "10.0.1.0/24",
	}
	resp, err := service.Create(t.Context(), req)
	if err != nil {
		t.Error(err)
	}
	if _, err := uuid.Parse(resp.Id); err != nil {
		t.Error("id of vpc could not be parsed into a UUID")
	}
}

func TestDelete(t *testing.T) {
	tests := make([]struct {
		vpc     *pb.Vpc
		request *pb.VpcIdentificationRequest
	}, 0, len(testVpcs)*len(identificationFunctions))

	for _, vpc := range testVpcs {
		for _, identificationFunction := range identificationFunctions {
			tests = append(tests, struct {
				vpc     *pb.Vpc
				request *pb.VpcIdentificationRequest
			}{vpc: vpc, request: identificationFunction(vpc)})
		}
	}

	for _, tt := range tests {
		service := NewVpcService(testVpcs)

		t.Run(tt.request.String(), func(t *testing.T) {
			_, err := service.Delete(t.Context(), tt.request)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := make([]struct {
		vpc     *pb.Vpc
		request *pb.VpcIdentificationRequest
	}, 0, len(testVpcs)*len(identificationFunctions))

	for _, vpc := range testVpcs {
		for _, identificationFunction := range identificationFunctions {
			tests = append(tests, struct {
				vpc     *pb.Vpc
				request *pb.VpcIdentificationRequest
			}{vpc: vpc, request: identificationFunction(vpc)})
		}
	}

	for _, tt := range tests {
		service := NewVpcService(testVpcs)

		t.Run(tt.request.String(), func(t *testing.T) {
			resp, err := service.Get(t.Context(), tt.request)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(tt.vpc, resp, protocmp.Transform()); diff != "" {
				t.Errorf("vpc mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestList(t *testing.T) {
	stream := &mockStream[*pb.Vpc]{}

	service := NewVpcService(testVpcs)
	service.List(&emptypb.Empty{}, stream)

	if len(testVpcs) != len(stream.SentItems) {
		t.Error("not the same amount of vpcs received")
	}

	for i := 0; i < len(testVpcs); i++ {
		if diff := cmp.Diff(testVpcs[i], stream.SentItems[i], protocmp.Transform()); diff != "" {
			t.Errorf("vpc mismatch (-want +got):\n%s", diff)
		}
	}
}
