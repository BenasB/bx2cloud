package handlers

import (
	"context"
	"testing"

	"github.com/BenasB/bx2cloud/internal/api"
)

func strPtr(s string) *string {
	return &s
}

func strTestName(s *string) string {
	if s == nil {
		return "nil"
	}

	if *s == "" {
		return "<empty>"
	}

	return *s
}

func TestGreet(t *testing.T) {
	tests := []struct {
		name *string
		out  string
	}{
		{nil, "Hello? I don't know your name"},
		{strPtr(""), "Hello from gRPC world to !"},
		{strPtr("Benas"), "Hello from gRPC world to Benas!"},
	}

	for _, tt := range tests {
		service := NewGreetService()

		t.Run(strTestName(tt.name), func(t *testing.T) {
			req := &api.GreetingRequest{
				Name: tt.name,
			}
			resp, err := service.Greet(context.Background(), req)
			if err != nil {
				t.Error(err)
			}
			if resp.Message != tt.out {
				t.Errorf("got %q, want %q", resp.Message, tt.out)
			}
		})
	}
}

func TestShoutGreet(t *testing.T) {
	tests := []struct {
		name *string
		out  string
	}{
		{nil, "Hello? I don't know your name"},
		{strPtr(""), "Hello from gRPC world to !"},
		{strPtr("Benas"), "Hello from gRPC world to BENAS!"},
	}

	for _, tt := range tests {
		service := NewGreetService()

		t.Run(strTestName(tt.name), func(t *testing.T) {
			req := &api.GreetingRequest{
				Name: tt.name,
			}
			resp, err := service.ShoutGreet(context.Background(), req)
			if err != nil {
				t.Error(err)
			}
			if resp.Message != tt.out {
				t.Errorf("got %q, want %q", resp.Message, tt.out)
			}
		})
	}
}
