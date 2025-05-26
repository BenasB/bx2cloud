package inputs

import (
	"fmt"
	"net"
)

var _ Input = &SubnetworkCreation{}

type SubnetworkCreation struct {
	Cidr string `yaml:"cidr"`
}

func (i *SubnetworkCreation) Validate() error {
	if i.Cidr == "" {
		return fmt.Errorf("missing required field: InternetAccess")
	}
	if _, _, err := net.ParseCIDR(i.Cidr); err != nil {
		return fmt.Errorf("Could not parse CIDR: %v", err)
	}
	return nil
}
