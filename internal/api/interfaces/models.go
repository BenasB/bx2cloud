package interfaces

import (
	"net"
	"os"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
)

type NetworkModel = pb.Network
type SubnetworkModel = pb.Subnetwork

type IpamType int

const (
	IPAM_UNALLOCATED IpamType = iota
	IPAM_CONTAINER
)

type ContainerModelData struct {
	Id                      uint32
	Ip                      *net.IPNet
	SubnetworkId            uint32
	Image                   string
	CreatedAt               time.Time
	StartedAt               time.Time
	EntrypointCustomization *ContainerProcessCustomization
	Spec                    *runspecs.Spec
}

type ContainerProcessCustomization struct {
	Entrypoint []string
	Cmd        []string
	Env        []string
}

type ContainerModel interface {
	GetData() *ContainerModelData
	GetState() (*runspecs.State, error)
	// Executes the user program in a 'created' container
	Exec() error
	Stop() error
	StartAdditionalProcess(process *runspecs.Process) (ContainerProcess, error)
}

type ContainerProcess interface {
	GetPty() *os.File
	Wait() (int, error)
	Stop() error
}

type ContainerCreationModel struct {
	Id                      uint32
	Ip                      *net.IPNet
	SubnetworkId            uint32
	Image                   string
	CreatedAt               time.Time
	EntrypointCustomization *ContainerProcessCustomization
	Spec                    *runspecs.Spec
}
