package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/BenasB/bx2cloud/internal/cli/common"
	"github.com/BenasB/bx2cloud/internal/cli/container"
	"github.com/BenasB/bx2cloud/internal/cli/exits"
	"github.com/BenasB/bx2cloud/internal/cli/introspection"
	"github.com/BenasB/bx2cloud/internal/cli/network"
	"github.com/BenasB/bx2cloud/internal/cli/subnetwork"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var globalFlagSet = flag.NewFlagSet("bx2cloud", flag.ExitOnError)
var globalFlags = struct {
	target *string
}{
	target: globalFlagSet.String("t", "localhost:8080", "API target <host>:<port>"),
}

func Run(args []string) exits.ExitCode {
	subcommands := make([]*common.CliCommand, 0)
	subcommands = append(subcommands, introspection.Commands...)
	subcommands = append(subcommands, network.Commands...)
	subcommands = append(subcommands, subnetwork.Commands...)
	subcommands = append(subcommands, container.Commands...)
	mainCommand := common.NewCliSubcommand(globalFlagSet.Name(), subcommands)

	globalFlagSet.Usage = func() {
		common.FprintSubcommands(os.Stderr, globalFlagSet.Name(), subcommands)

		fmt.Fprintf(os.Stderr, "%s flags:\n", globalFlagSet.Name())
		globalFlagSet.PrintDefaults()
	}
	if err := globalFlagSet.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exits.BAD_FLAG
	}

	conn, err := newConn(*globalFlags.target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exits.BAD_FLAG
	}
	defer conn.Close()

	return mainCommand.Execute(globalFlagSet.Args(), conn, []string{})
}

func newConn(target string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
