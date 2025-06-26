package container

import (
	"github.com/BenasB/bx2cloud/internal/api/shared"
)

type configurator interface {
	Configure(model shared.ContainerModel, subnetworkModel *shared.SubnetworkModel) error
	Unconfigure(model shared.ContainerModel, subnetworkModel *shared.SubnetworkModel) error
}
