package main

import (
	"context"
	"fmt"

	"github.com/BenasB/bx2cloud/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.NewClient("localhost:8080", opts...)
	if err != nil {
		fmt.Printf("failed to contact the server: %s\n", err)
		return
	}
	defer conn.Close()

	client := api.NewApiClient(conn)

	resp, err := client.Greet(context.Background(), &api.GreetingRequest{})
	if err != nil {
		fmt.Println("failed to greet")
		return
	}

	fmt.Println(resp)
}
