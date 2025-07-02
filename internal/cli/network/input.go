package network

import "github.com/BenasB/bx2cloud/internal/cli/inputs"

var _ inputs.Input = &networkCreation{}

type networkCreation struct {
	InternetAccess bool `yaml:"internetAccess"`
}

func (i *networkCreation) Validate() error {
	return nil
}
