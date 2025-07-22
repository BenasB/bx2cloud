package logs

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
)

var _ Logger = &fsLogger{}

type fsLogger struct {
	root string
}

func NewFsLogger() (Logger, error) {
	root := "/var/log/bx2cloud"
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}

	return &fsLogger{
		root: root,
	}, nil
}

func (l *fsLogger) Init(containerId uint32) (*os.File, error) {
	idString := strconv.FormatInt(int64(containerId), 10)
	return os.OpenFile(filepath.Join(l.root, idString), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
}

func (l *fsLogger) Get(containerId uint32) (io.ReadCloser, error) {
	idString := strconv.FormatInt(int64(containerId), 10)
	return os.Open(filepath.Join(l.root, idString))
}
