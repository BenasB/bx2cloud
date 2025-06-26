package shared

import (
	"net"

	"github.com/opencontainers/runc/libcontainer"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
)

type ContainerModel = libcontainer.Container

type ContainerCreationModel struct {
	Id           uint32
	Ip           *net.IPNet
	SubnetworkId uint32
	Image        string
	Spec         *runspecs.Spec
}
