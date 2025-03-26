package cli

import (
	"context"
	"fmt"

	pb "github.com/BenasB/bx2cloud/internal/api"
)

func greet(client pb.GreetServiceClient) error {
	resp, err := client.Greet(context.Background(), &pb.GreetingRequest{})
	if err != nil {
		return err
	}

	fmt.Println(resp)

	return nil
}

func greetName(client pb.GreetServiceClient, name string) error {
	resp, err := client.Greet(context.Background(), &pb.GreetingRequest{Name: &name})
	if err != nil {
		return err
	}

	fmt.Println(resp)

	return nil
}
