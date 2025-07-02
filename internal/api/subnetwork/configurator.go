package subnetwork

import "github.com/BenasB/bx2cloud/internal/api/interfaces"

type configurator interface {
	Configure(model *interfaces.SubnetworkModel) error
	Unconfigure(model *interfaces.SubnetworkModel) error
}

var _ configurator = &mockConfigurator{}

type mockConfigurator struct{}

func NewMockConfigurator() configurator {
	return &mockConfigurator{}
}

func (m *mockConfigurator) Configure(model *interfaces.SubnetworkModel) error {
	return nil
}

func (m *mockConfigurator) Unconfigure(model *interfaces.SubnetworkModel) error {
	return nil
}
