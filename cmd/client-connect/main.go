// Клиент на ConnectRPC: оборачивает стандартный http.Client.
// Флагом протокола можно заставить его говорить по Connect, gRPC или gRPC-Web —
// с одним и тем же сервером.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"

	greetv1 "github.com/artyomtrofimov/grpc-connectrpc-demo/gen/greet/v1"
	"github.com/artyomtrofimov/grpc-connectrpc-demo/gen/greet/v1/greetv1connect"
)

func main() {
	protocol := flag.String("protocol", "connect", "протокол: connect | grpc | grpcweb")
	baseURL := flag.String("url", "http://localhost:8080", "адрес сервера")
	flag.Parse()

	var opts []connect.ClientOption
	// Connect и gRPC-Web живут и на обычном HTTP/1.1,
	// а вот gRPC-протоколу нужен HTTP/2 (без TLS — это h2c).
	httpClient := http.DefaultClient

	switch *protocol {
	case "connect":
		// протокол по умолчанию, опции не нужны
	case "grpc":
		opts = append(opts, connect.WithGRPC())
		httpClient = newH2CClient()
	case "grpcweb":
		opts = append(opts, connect.WithGRPCWeb())
	default:
		log.Fatalf("неизвестный протокол: %s", *protocol)
	}

	client := greetv1connect.NewGreetServiceClient(httpClient, *baseURL, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Unary
	resp, err := client.Greet(ctx, connect.NewRequest(&greetv1.GreetRequest{Name: "Хабр"}))
	if err != nil {
		log.Fatalf("greet: %v", err)
	}
	log.Printf("[%s] unary: %s", *protocol, resp.Msg.GetGreeting())

	// Server streaming
	stream, err := client.GreetStream(ctx, connect.NewRequest(&greetv1.GreetStreamRequest{Name: "Хабр", Count: 3}))
	if err != nil {
		log.Fatalf("greet stream: %v", err)
	}
	for stream.Receive() {
		msg := stream.Msg()
		log.Printf("[%s] stream #%d: %s", *protocol, msg.GetSequence(), msg.GetGreeting())
	}
	if err := stream.Err(); err != nil {
		log.Fatalf("stream: %v", err)
	}
}

// newH2CClient возвращает http.Client, говорящий по HTTP/2 без TLS (h2c).
// С Go 1.24 это штатная возможность net/http.
func newH2CClient() *http.Client {
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	return &http.Client{
		Transport: &http.Transport{Protocols: &protocols},
	}
}
