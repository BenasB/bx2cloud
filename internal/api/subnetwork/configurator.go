package subnetwork

import (
	"github.com/BenasB/bx2cloud/internal/api/shared"
)

type configurator interface {
	Configure(model *shared.SubnetworkModel) error
	Unconfigure(model *shared.SubnetworkModel) error
}

var _ configurator = &mockConfigurator{}

type mockConfigurator struct{}

func NewMockConfigurator() configurator {
	return &mockConfigurator{}
}

func (m *mockConfigurator) Configure(model *shared.SubnetworkModel) error {
	return nil
}

func (m *mockConfigurator) Unconfigure(model *shared.SubnetworkModel) error {
	return nil
}
