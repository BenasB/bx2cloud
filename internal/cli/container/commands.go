package container

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
		"container",
		[]*common.CliCommand{
			common.NewCliCommand(
				"list",
				"Retrieves all existing containers",
				"",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)
					if err := List(client); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"get",
				"Retrieves a specified container",
				"<id>",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)
					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Get(client, id); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"delete",
				"Deletes a specified container. Before that, stops it if it is running.",
				"<id>",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)
					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Delete(client, id); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"create",
				"Creates and starts a new container resource",
				"< file.yaml",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)

					yamlBytes, err := io.ReadAll(os.Stdin)
					if err != nil {
						return exits.CONTAINER_ERROR, err
					}

					if err := Create(client, yamlBytes); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"exec",
				"Starts a shell process inside a specified container or executes a specific command, if specified",
				"<id> [cmd]",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)
					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Exec(client, id, args); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"start",
				"Starts a specified container resource",
				"<id>",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)
					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Start(client, id); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"stop",
				"Stops a specified container resource",
				"<id>",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)
					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Stop(client, id); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
			common.NewCliCommand(
				"logs",
				"Retrieves the logs of a specified container resource",
				"<id>",
				func(args []string, conn *grpc.ClientConn) (exits.ExitCode, error) {
					client := pb.NewContainerServiceClient(conn)
					id, exitCode, err := common.ParseUint32Arg(&args)
					if err != nil {
						return exitCode, fmt.Errorf("failed to parse 'id' argument: %w", err)
					}

					if err := Logs(client, id); err != nil {
						return exits.CONTAINER_ERROR, err
					}
					return exits.SUCCESS, nil
				},
			),
		},
	),
}
