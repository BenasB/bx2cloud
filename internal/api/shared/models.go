package shared

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
	Id           uint32
	Ip           *net.IPNet
	SubnetworkId uint32
	Image        string
	CreatedAt    time.Time
}

type ContainerModel interface {
	GetData() *ContainerModelData
	GetState() (*runspecs.State, error)
	Exec() error
	StartInteractive(process *runspecs.Process) (ContainerInteractiveProcess, error)
	Signal(os.Signal) error
}

type ContainerInteractiveProcess interface {
	GetPty() *os.File
	Wait() (int, error)
	Signal(os.Signal) error
}

type ContainerCreationModel struct {
	Id           uint32
	Ip           *net.IPNet
	SubnetworkId uint32
	Image        string
	Spec         *runspecs.Spec
}
