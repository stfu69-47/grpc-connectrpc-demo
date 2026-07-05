// Бенчмарки: grpc-go против connect-go на одном и том же сервисе.
// Оба сервера поднимаются на реальном TCP (localhost, случайный порт),
// без TLS; connect — через h2c, чтобы оба стека работали по HTTP/2.
// Интерцепторы не подключаются — меряем чистый транспорт.
package bench

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	greetv1 "github.com/artyomtrofimov/grpc-connectrpc-demo/gen/greet/v1"
	"github.com/artyomtrofimov/grpc-connectrpc-demo/gen/greet/v1/greetv1connect"
	"github.com/artyomtrofimov/grpc-connectrpc-demo/internal/greeter"
)

const (
	smallPayload = 100
	largePayload = 100 * 1024
	streamCount  = 100
)

// ---- реализации сервиса (без логирования) ----

type grpcServer struct {
	greetv1.UnimplementedGreetServiceServer
}

func (s *grpcServer) Greet(ctx context.Context, req *greetv1.GreetRequest) (*greetv1.GreetResponse, error) {
	return &greetv1.GreetResponse{Greeting: greeter.Greet(req.GetName())}, nil
}

func (s *grpcServer) GreetStream(req *greetv1.GreetStreamRequest, stream grpc.ServerStreamingServer[greetv1.GreetStreamResponse]) error {
	for i := int32(1); i <= req.GetCount(); i++ {
		if err := stream.Send(&greetv1.GreetStreamResponse{
			Greeting: greeter.GreetSeq(req.GetName(), i),
			Sequence: i,
		}); err != nil {
			return err
		}
	}
	return nil
}

type connectServer struct{}

func (s *connectServer) Greet(ctx context.Context, req *connect.Request[greetv1.GreetRequest]) (*connect.Response[greetv1.GreetResponse], error) {
	return connect.NewResponse(&greetv1.GreetResponse{Greeting: greeter.Greet(req.Msg.GetName())}), nil
}

func (s *connectServer) GreetStream(ctx context.Context, req *connect.Request[greetv1.GreetStreamRequest], stream *connect.ServerStream[greetv1.GreetStreamResponse]) error {
	for i := int32(1); i <= req.Msg.GetCount(); i++ {
		if err := stream.Send(&greetv1.GreetStreamResponse{
			Greeting: greeter.GreetSeq(req.Msg.GetName(), i),
			Sequence: i,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ---- запуск серверов ----

func startGRPCServer(b *testing.B) string {
	b.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	srv := grpc.NewServer()
	greetv1.RegisterGreetServiceServer(srv, &grpcServer{})
	go func() { _ = srv.Serve(lis) }()
	b.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func startConnectServer(b *testing.B) string {
	b.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	mux := http.NewServeMux()
	path, handler := greetv1connect.NewGreetServiceHandler(&connectServer{})
	mux.Handle(path, handler)
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	srv := &http.Server{Handler: mux, Protocols: &protocols}
	go func() { _ = srv.Serve(lis) }()
	b.Cleanup(func() { _ = srv.Close() })
	return "http://" + lis.Addr().String()
}

// ---- клиенты ----

func grpcClient(b *testing.B, addr string) greetv1.GreetServiceClient {
	b.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = conn.Close() })
	return greetv1.NewGreetServiceClient(conn)
}

func h2cHTTPClient() *http.Client {
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	return &http.Client{
		Transport: &http.Transport{Protocols: &protocols},
	}
}

func connectClient(b *testing.B, baseURL string, opts ...connect.ClientOption) greetv1connect.GreetServiceClient {
	b.Helper()
	return greetv1connect.NewGreetServiceClient(h2cHTTPClient(), baseURL, opts...)
}

// ---- unary ----

func benchGRPCUnary(b *testing.B, payload int) {
	client := grpcClient(b, startGRPCServer(b))
	name := strings.Repeat("a", payload)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Greet(ctx, &greetv1.GreetRequest{Name: name}); err != nil {
			b.Fatal(err)
		}
	}
}

func benchConnectUnary(b *testing.B, payload int, opts ...connect.ClientOption) {
	client := connectClient(b, startConnectServer(b), opts...)
	name := strings.Repeat("a", payload)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Greet(ctx, connect.NewRequest(&greetv1.GreetRequest{Name: name})); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnary_GRPCGo_Proto_100B(b *testing.B)  { benchGRPCUnary(b, smallPayload) }
func BenchmarkUnary_GRPCGo_Proto_100KB(b *testing.B) { benchGRPCUnary(b, largePayload) }

func BenchmarkUnary_Connect_Proto_100B(b *testing.B)  { benchConnectUnary(b, smallPayload) }
func BenchmarkUnary_Connect_Proto_100KB(b *testing.B) { benchConnectUnary(b, largePayload) }

func BenchmarkUnary_ConnectGRPCProto_Proto_100B(b *testing.B) {
	benchConnectUnary(b, smallPayload, connect.WithGRPC())
}

func BenchmarkUnary_Connect_JSON_100B(b *testing.B) {
	benchConnectUnary(b, smallPayload, connect.WithProtoJSON())
}
func BenchmarkUnary_Connect_JSON_100KB(b *testing.B) {
	benchConnectUnary(b, largePayload, connect.WithProtoJSON())
}

// ---- server streaming (100 сообщений по ~100 байт за операцию) ----

func BenchmarkStream_GRPCGo(b *testing.B) {
	client := grpcClient(b, startGRPCServer(b))
	name := strings.Repeat("a", smallPayload)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.GreetStream(ctx, &greetv1.GreetStreamRequest{Name: name, Count: streamCount})
		if err != nil {
			b.Fatal(err)
		}
		for {
			if _, err := stream.Recv(); err != nil {
				break
			}
		}
	}
}

func BenchmarkStream_Connect(b *testing.B) {
	client := connectClient(b, startConnectServer(b))
	name := strings.Repeat("a", smallPayload)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.GreetStream(ctx, connect.NewRequest(&greetv1.GreetStreamRequest{Name: name, Count: streamCount}))
		if err != nil {
			b.Fatal(err)
		}
		for stream.Receive() {
		}
		if err := stream.Err(); err != nil {
			b.Fatal(err)
		}
		_ = stream.Close()
	}
}
