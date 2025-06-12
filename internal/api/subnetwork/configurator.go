package subnetwork

import (
	"github.com/BenasB/bx2cloud/internal/api/shared"
)

type configurator interface {
	configure(model *shared.SubnetworkModel) error
	unconfigure(model *shared.SubnetworkModel) error
}

var _ configurator = &mockConfigurator{}

type mockConfigurator struct{}

func NewMockConfigurator() configurator {
	return &mockConfigurator{}
}

func (m *mockConfigurator) configure(model *shared.SubnetworkModel) error {
	return nil
}

func (m *mockConfigurator) unconfigure(model *shared.SubnetworkModel) error {
	return nil
}
