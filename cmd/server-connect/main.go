// ConnectRPC-сервер: обычный http.Handler, из коробки отвечает
// на три протокола — Connect, gRPC и gRPC-Web.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"

	greetv1 "github.com/stfu69-47/grpc-connectrpc-demo/gen/greet/v1"
	"github.com/stfu69-47/grpc-connectrpc-demo/gen/greet/v1/greetv1connect"
	"github.com/stfu69-47/grpc-connectrpc-demo/internal/greeter"
	"github.com/stfu69-47/grpc-connectrpc-demo/internal/logging"
)

const addr = ":8080"

type greetServer struct{}

func (s *greetServer) Greet(
	ctx context.Context,
	req *connect.Request[greetv1.GreetRequest],
) (*connect.Response[greetv1.GreetResponse], error) {
	return connect.NewResponse(&greetv1.GreetResponse{
		Greeting: greeter.Greet(req.Msg.GetName()),
	}), nil
}

func (s *greetServer) GreetStream(
	ctx context.Context,
	req *connect.Request[greetv1.GreetStreamRequest],
	stream *connect.ServerStream[greetv1.GreetStreamResponse],
) error {
	for i := int32(1); i <= req.Msg.GetCount(); i++ {
		resp := &greetv1.GreetStreamResponse{
			Greeting: greeter.GreetSeq(req.Msg.GetName(), i),
			Sequence: i,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	mux := http.NewServeMux()

	// Connect-хендлер монтируется в обычный ServeMux —
	// рядом можно повесить health check, метрики, любые HTTP-эндпоинты.
	path, handler := greetv1connect.NewGreetServiceHandler(
		&greetServer{},
		connect.WithInterceptors(logging.ConnectLogging{}),
	)
	mux.Handle(path, handler)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// HTTP/2 без TLS (h2c) нужен, чтобы классические gRPC-клиенты
	// могли ходить на этот порт без сертификатов; протокол Connect
	// работает и по обычному HTTP/1.1. С Go 1.24 это включается
	// штатно, без golang.org/x/net.
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		Protocols:         &protocols,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Connect-сервер слушает %s (Connect + gRPC + gRPC-Web)", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
