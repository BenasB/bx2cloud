package cli

import (
	"fmt"
	"io"
	"os"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/cli/exits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO: use flags package

func Run(args []string) exits.ExitCode {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Missing command\n")
		return exits.MISSING_COMMAND
	}

	command := args[0]
	args = args[1:]

	conn, err := newConn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exits.SERVER_ERROR
	}
	defer conn.Close()

	var cmdErrCode exits.ExitCode
	var cmdErr error
	switch command {
	case "vpc":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Missing subcommand\n")
			return exits.MISSING_SUBCOMMAND
		}
		subcommand := args[0]
		args = args[1:]

		client := pb.NewVpcServiceClient(conn)
		cmdErrCode = exits.VPC_ERROR

		switch subcommand {
		case "list":
			cmdErr = vpcList(client)
		case "get":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing identifier argument (id or name)\n")
				return exits.MISSING_ARGUMENT
			}

			identifier := args[0]
			args = args[1:]
			cmdErr = vpcGet(client, identifier)
		case "delete":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing identifier argument (id or name)\n")
				return exits.MISSING_ARGUMENT
			}

			identifier := args[0]
			args = args[1:]
			cmdErr = vpcDelete(client, identifier)
		case "create":
			yamlBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				cmdErr = err
				break
			}

			cmdErr = vpcCreate(client, yamlBytes)
		default:
			fmt.Fprintf(os.Stderr, "Unrecognized subcommand '%s'\n", subcommand)
			return exits.UNKNOWN_SUBCOMMAND
		}
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized command '%s'\n", command)
		return exits.UNKNOWN_COMMAND
	}

	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", cmdErr)
		return cmdErrCode
	}

	return exits.SUCCESS
}

func newConn() (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.NewClient("localhost:8080", opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
