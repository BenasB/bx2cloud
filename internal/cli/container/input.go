package container

import (
	"fmt"

	"github.com/BenasB/bx2cloud/internal/cli/inputs"
)

var _ inputs.Input = &containerCreation{}

type containerCreation struct {
	SubnetworkId uint32   `yaml:"subnetworkId"`
	Image        string   `yaml:"image"`
	Entrypoint   []string `yaml:"entrypoint"`
	Cmd          []string `yaml:"cmd"`
	Env          []string `yaml:"env"`
}

func (i *containerCreation) Validate() error {
	if i.SubnetworkId == 0 {
		return fmt.Errorf("missing required field: subnetworkId")
	}
	if i.Image == "" {
		return fmt.Errorf("missing required field: image")
	}
	return nil
}
