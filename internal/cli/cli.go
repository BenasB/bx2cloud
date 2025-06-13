package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/cli/container"
	"github.com/BenasB/bx2cloud/internal/cli/exits"
	"github.com/BenasB/bx2cloud/internal/cli/network"
	"github.com/BenasB/bx2cloud/internal/cli/subnetwork"
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
			cmdErr = network.List(client)
		case "get":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = network.Get(client, uint32(id))
		case "delete":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = network.Delete(client, uint32(id))
		case "create":
			// TODO: Read a file and fallback to os.Stdin if no file is supplied
			yamlBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				cmdErr = err
				break
			}

			cmdErr = network.Create(client, yamlBytes)
		case "update":
			yamlBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				cmdErr = err
				break
			}

			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}

			cmdErr = network.Update(client, uint32(id), yamlBytes)
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
			cmdErr = subnetwork.List(client)
		case "get":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = subnetwork.Get(client, uint32(id))
		case "delete":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = subnetwork.Delete(client, uint32(id))
		case "create":
			yamlBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				cmdErr = err
				break
			}

			cmdErr = subnetwork.Create(client, yamlBytes)
		case "update":
			yamlBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				cmdErr = err
				break
			}

			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}

			cmdErr = subnetwork.Update(client, uint32(id), yamlBytes)
		default:
			fmt.Fprintf(os.Stderr, "Unrecognized subcommand '%s'\n", subcommand)
			return exits.UNKNOWN_SUBCOMMAND
		}
	case "container":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Missing subcommand\n")
			return exits.MISSING_SUBCOMMAND
		}
		subcommand := args[0]
		args = args[1:]

		client := pb.NewContainerServiceClient(conn)
		cmdErrCode = exits.CONTAINER_ERROR

		switch subcommand {
		case "list":
			cmdErr = container.List(client)
		case "get":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = container.Get(client, uint32(id))
		case "delete":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = container.Delete(client, uint32(id))
		case "create":
			cmdErr = container.Create(client)
		case "exec":
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "Missing id argument\n")
				return exits.MISSING_ARGUMENT
			}

			idString := args[0]
			args = args[1:]

			id, err := strconv.ParseUint(idString, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not convert the id '%d' argument to an integer\n", id)
				return exits.BAD_ARGUMENT
			}
			cmdErr = container.Exec(client, uint32(id))
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
