// Package logging — интерцепторы логирования для обоих стеков.
package logging

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/grpc"
)

// GRPCUnaryLogging — серверный unary-интерцептор для grpc-go.
// Для стриминга, а также клиентской стороны в grpc-go нужны
// отдельные реализации с другими сигнатурами.
func GRPCUnaryLogging(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("grpc call=%s dur=%s err=%v", info.FullMethod, time.Since(start), err)
	return resp, err
}

// ConnectLogging — единый интерцептор для connect-go: одна реализация
// покрывает unary и стриминг, сервер и клиент.
type ConnectLogging struct{}

func (ConnectLogging) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		resp, err := next(ctx, req)
		log.Printf("call=%s protocol=%s dur=%s err=%v",
			req.Spec().Procedure, req.Peer().Protocol, time.Since(start), err)
		return resp, err
	}
}

func (ConnectLogging) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		log.Printf("stream client call=%s", spec.Procedure)
		return next(ctx, spec)
	}
}

func (ConnectLogging) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		err := next(ctx, conn)
		log.Printf("stream call=%s protocol=%s dur=%s err=%v",
			conn.Spec().Procedure, conn.Peer().Protocol, time.Since(start), err)
		return err
	}
}
