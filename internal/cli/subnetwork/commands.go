package subnetwork

import (
	"fmt"
	"io"
	"os"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/cli/common"
	"github.com/BenasB/bx2cloud/internal/cli/exits"
	"google.golang.org/grpc"
)

var Commands = []*common.CliCommand{
	common.NewCliSubcommand(
		"subnetwork",
		[]*common.CliCommand{
			common.NewCliCommand(
				"list",
				"Retrieves all existing subnetworks",
				"",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewSubnetworkServiceClient(conn)
					if err := List(client); err != nil {
						return exits.SUBNETWORK_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"get",
				"Retrieves a specified subnetwork",
				"<id>",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewSubnetworkServiceClient(conn)
					if err := List(client); err != nil {
						return exits.SUBNETWORK_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"delete",
				"Deletes a specified subnetwork",
				"<id>",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewSubnetworkServiceClient(conn)
					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Delete(client, id); err != nil {
						return exits.SUBNETWORK_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"create",
				"Creates a new subnetwork resource",
				"< file.yaml",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewSubnetworkServiceClient(conn)

					yamlBytes, err := io.ReadAll(os.Stdin)
					if err != nil {
						return exits.SUBNETWORK_ERROR, err
					}

					if err := Create(client, yamlBytes); err != nil {
						return exits.SUBNETWORK_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"update",
				"Updates an existing network resource",
				"< file.yaml",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewSubnetworkServiceClient(conn)

					yamlBytes, err := io.ReadAll(os.Stdin)
					if err != nil {
						return exits.SUBNETWORK_ERROR, err
					}

					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Update(client, id, yamlBytes); err != nil {
						return exits.SUBNETWORK_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
		},
	),
}
