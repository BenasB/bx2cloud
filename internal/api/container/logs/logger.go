package logs

import (
	"os"
)

type Logger interface {
	Init(containerId uint32) (*os.File, error)
	Remove(containerId uint32) error
	Get(containerId uint32) (*os.File, error)
}
