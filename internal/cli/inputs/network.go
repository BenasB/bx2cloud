package inputs

var _ Input = &NetworkCreation{}

type NetworkCreation struct {
	InternetAccess bool `yaml:"internetAccess"`
}

func (i *NetworkCreation) Validate() error {
	return nil
}
