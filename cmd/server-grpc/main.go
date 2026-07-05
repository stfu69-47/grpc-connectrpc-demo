// Классический gRPC-сервер на grpc-go.
package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	greetv1 "github.com/artyomtrofimov/grpc-connectrpc-demo/gen/greet/v1"
	"github.com/artyomtrofimov/grpc-connectrpc-demo/internal/greeter"
	"github.com/artyomtrofimov/grpc-connectrpc-demo/internal/logging"
)

const addr = ":50051"

type greetServer struct {
	greetv1.UnimplementedGreetServiceServer
}

func (s *greetServer) Greet(ctx context.Context, req *greetv1.GreetRequest) (*greetv1.GreetResponse, error) {
	return &greetv1.GreetResponse{Greeting: greeter.Greet(req.GetName())}, nil
}

func (s *greetServer) GreetStream(req *greetv1.GreetStreamRequest, stream grpc.ServerStreamingServer[greetv1.GreetStreamResponse]) error {
	for i := int32(1); i <= req.GetCount(); i++ {
		resp := &greetv1.GreetStreamResponse{
			Greeting: greeter.GreetSeq(req.GetName(), i),
			Sequence: i,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logging.GRPCUnaryLogging),
	)
	greetv1.RegisterGreetServiceServer(srv, &greetServer{})
	// Рефлексия — чтобы работал grpcurl без указания .proto-файлов.
	reflection.Register(srv)

	log.Printf("gRPC-сервер слушает %s", addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
