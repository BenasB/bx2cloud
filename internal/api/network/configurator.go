package network

import (
	"github.com/BenasB/bx2cloud/internal/api/shared"
)

type configurator interface {
	configure(model *shared.NetworkModel) error
	unconfigure(model *shared.NetworkModel) error
}

var _ configurator = &mockConfigurator{}

type mockConfigurator struct{}

func NewMockConfigurator() configurator {
	return &mockConfigurator{}
}

func (m *mockConfigurator) configure(model *shared.NetworkModel) error {
	return nil
}

func (m *mockConfigurator) unconfigure(model *shared.NetworkModel) error {
	return nil
}
