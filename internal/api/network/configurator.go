package network

import "github.com/BenasB/bx2cloud/internal/api/interfaces"

type configurator interface {
	Configure(model *interfaces.NetworkModel) error
	Unconfigure(model *interfaces.NetworkModel) error
}

var _ configurator = &mockConfigurator{}

type mockConfigurator struct{}

func NewMockConfigurator() configurator {
	return &mockConfigurator{}
}

func (m *mockConfigurator) Configure(model *interfaces.NetworkModel) error {
	return nil
}

func (m *mockConfigurator) Unconfigure(model *interfaces.NetworkModel) error {
	return nil
}
