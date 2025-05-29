package shared

import (
	"context"

	"google.golang.org/grpc"
)

type mockStream[T any] struct {
	grpc.ServerStream
	SentItems []T
	ctx       context.Context
}

func (s *mockStream[T]) Send(item T) error {
	s.SentItems = append(s.SentItems, item)
	return nil
}

func (s *mockStream[T]) Context() context.Context {
	return s.ctx
}

func NewMockStream[T any](ctx context.Context) *mockStream[T] {
	return &mockStream[T]{
		ctx: ctx,
	}
}
