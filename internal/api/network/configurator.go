package network

import (
	"github.com/BenasB/bx2cloud/internal/api/shared"
)

type configurator interface {
	Configure(model *shared.NetworkModel) error
	Unconfigure(model *shared.NetworkModel) error
}

var _ configurator = &mockConfigurator{}

type mockConfigurator struct{}

func NewMockConfigurator() configurator {
	return &mockConfigurator{}
}

func (m *mockConfigurator) Configure(model *shared.NetworkModel) error {
	return nil
}

func (m *mockConfigurator) Unconfigure(model *shared.NetworkModel) error {
	return nil
}
