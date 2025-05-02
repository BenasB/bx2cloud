package inputs

import "fmt"

type VpcCreation struct {
	Name string `yaml:"name"`
	Cidr string `yaml:"cidr"`
}

func (i *VpcCreation) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if i.Cidr == "" {
		return fmt.Errorf("missing required field: cidr")
	}
	return nil
}
