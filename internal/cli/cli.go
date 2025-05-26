package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

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
	case "network":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Missing subcommand\n")
			return exits.MISSING_SUBCOMMAND
		}
		subcommand := args[0]
		args = args[1:]

		client := pb.NewNetworkServiceClient(conn)
		cmdErrCode = exits.NETWORK_ERROR

		switch subcommand {
		case "list":
			cmdErr = networkList(client)
		case "get":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%s' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = networkGet(client, uint32(id))
		case "delete":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%s' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = networkDelete(client, uint32(id))
		case "create":
			yamlBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				cmdErr = err
				break
			}

			cmdErr = networkCreate(client, yamlBytes)
		default:
			fmt.Fprintf(os.Stderr, "Unrecognized subcommand '%s'\n", subcommand)
			return exits.UNKNOWN_SUBCOMMAND
		}
	case "subnetwork":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Missing subcommand\n")
			return exits.MISSING_SUBCOMMAND
		}
		subcommand := args[0]
		args = args[1:]

		client := pb.NewSubnetworkServiceClient(conn)
		cmdErrCode = exits.SUBNETWORK_ERROR

		switch subcommand {
		case "list":
			cmdErr = subnetworkList(client)
		case "get":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%s' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = subnetworkGet(client, uint32(id))
		case "delete":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%s' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = subnetworkDelete(client, uint32(id))
		case "create":
			yamlBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				cmdErr = err
				break
			}

			cmdErr = subnetworkCreate(client, yamlBytes)
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
