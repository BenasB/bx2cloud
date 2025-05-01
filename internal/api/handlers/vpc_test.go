package handlers

import (
	"testing"
	"time"

	"github.com/BenasB/bx2cloud/internal/api"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCreate(t *testing.T) {
	service := NewVpcService(make([]*api.Vpc, 0))
	t.Run("Create", func(t *testing.T) {
		req := &api.VpcCreationRequest{
			Name: "my-vpc",
			Cidr: "10.0.1.0/24",
		}
		resp, err := service.Create(t.Context(), req)
		if err != nil {
			t.Error(err)
		}
		if _, err := uuid.Parse(resp.Id); err != nil {
			t.Error("Id of VPC could not be parsed into a UUID")
		}
	})
}

func TestDelete(t *testing.T) {
	existingVpcs := make([]*api.Vpc, 0)

	existingVpcs = append(existingVpcs, &api.Vpc{
		Id:        "abc-f12",
		Name:      "first-vpc",
		Cidr:      "10.0.1.0/24",
		CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
	})

	existingVpcs = append(existingVpcs, &api.Vpc{
		Id:        "def-h41x21",
		Name:      "second-vpc",
		Cidr:      "10.0.2.0/24",
		CreatedAt: timestamppb.New(time.Now().Add(-time.Minute)),
	})

	tests := []*api.VpcIdentificationRequest{
		{
			Identification: &api.VpcIdentificationRequest_Id{
				Id: existingVpcs[0].Id,
			},
		},
		{
			Identification: &api.VpcIdentificationRequest_Id{
				Id: existingVpcs[1].Id,
			},
		},
		{
			Identification: &api.VpcIdentificationRequest_Name{
				Name: existingVpcs[0].Name,
			},
		},
		{
			Identification: &api.VpcIdentificationRequest_Name{
				Name: existingVpcs[1].Name,
			},
		},
	}

	for _, tt := range tests {
		service := NewVpcService(existingVpcs)

		t.Run(tt.String(), func(t *testing.T) {
			_, err := service.Delete(t.Context(), tt)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
