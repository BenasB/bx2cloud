package introspection

import (
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/cli/common"
	"github.com/BenasB/bx2cloud/internal/cli/exits"
	"google.golang.org/grpc"
)

var Commands = []*common.CliCommand{
	common.NewCliCommand(
		"version",
		"Prints out CLI and API version information",
		func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
			client := pb.NewIntrospectionServiceClient(conn)
			Version(client)
			return exits.SUCCESS, nil
		},
	),
}
