package container

import "github.com/BenasB/bx2cloud/internal/api/interfaces"

type configurator interface {
	Configure(model interfaces.ContainerModel, subnetworkModel *interfaces.SubnetworkModel) error
	Unconfigure(model interfaces.ContainerModel, subnetworkModel *interfaces.SubnetworkModel) error
}
