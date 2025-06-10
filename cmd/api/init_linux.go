// Used by libcontainer to initialize the container
// see https://github.com/opencontainers/runc/blob/main/init.go

package main

import (
	"os"

	"github.com/opencontainers/runc/libcontainer"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		libcontainer.Init()
	}
}
