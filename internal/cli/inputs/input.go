package inputs

type Input interface {
	Validate() error
}
