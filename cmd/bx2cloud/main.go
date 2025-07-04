package main

import (
	"os"

	"github.com/BenasB/bx2cloud/internal/cli"
)

func main() {
	exitCode := int(cli.Run(os.Args[1:]))
	os.Exit(exitCode)
}
