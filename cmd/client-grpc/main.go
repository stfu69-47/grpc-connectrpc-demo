// Клиент на классическом grpc-go.
package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	greetv1 "github.com/stfu69-47/grpc-connectrpc-demo/gen/greet/v1"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "адрес сервера")
	flag.Parse()

	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	client := greetv1.NewGreetServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Unary
	resp, err := client.Greet(ctx, &greetv1.GreetRequest{Name: "Хабр"})
	if err != nil {
		log.Fatalf("greet: %v", err)
	}
	log.Printf("unary: %s", resp.GetGreeting())

	// Server streaming
	stream, err := client.GreetStream(ctx, &greetv1.GreetStreamRequest{Name: "Хабр", Count: 3})
	if err != nil {
		log.Fatalf("greet stream: %v", err)
	}
	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalf("stream recv: %v", err)
		}
		log.Printf("stream #%d: %s", msg.GetSequence(), msg.GetGreeting())
	}
}
