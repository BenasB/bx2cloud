package subnetwork

import (
	"fmt"
	"net"

	"github.com/BenasB/bx2cloud/internal/cli/inputs"
)

var _ inputs.Input = &subnetworkCreation{}

type subnetworkCreation struct {
	NetworkId uint32 `yaml:"networkId"`
	Cidr      string `yaml:"cidr"`
}

func (i *subnetworkCreation) Validate() error {
	// TODO: handle missing vs default value
	if i.NetworkId == 0 {
		return fmt.Errorf("missing required field: networkId")
	}
	if i.Cidr == "" {
		return fmt.Errorf("missing required field: cidr")
	}
	if _, _, err := net.ParseCIDR(i.Cidr); err != nil {
		return fmt.Errorf("Could not parse CIDR: %v", err)
	}
	return nil
}
