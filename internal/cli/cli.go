package cli

import (
	"fmt"
	"os"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/cli/exits"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO: use flags package

func Run(args []string) exits.ExitCode {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Missing command\n")
		return exits.MISSING_COMMAND
	}

	command := args[1]

	conn, err := newConn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return exits.SERVER_ERROR
	}
	defer conn.Close()

	var cmdErrCode exits.ExitCode
	var cmdErr error
	switch command {
	case "greet":
		cmdErr, cmdErrCode = greet(pb.NewGreetServiceClient(conn)), exits.GREET_ERROR
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized command '%s'\n", command)
		return exits.UNKNOWN_COMMAND
	}

	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cmdErr)
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
