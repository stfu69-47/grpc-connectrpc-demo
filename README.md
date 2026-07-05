# gRPC vs ConnectRPC — демо

Демо-проект к статье на Хабре: один и тот же сервис `GreetService`
(unary + server streaming), реализованный на классическом
[grpc-go](https://github.com/grpc/grpc-go) и на
[ConnectRPC](https://connectrpc.com).

## Структура

```
proto/greet/v1/greet.proto   — Protobuf-контракт (единый для обоих серверов)
gen/                         — сгенерированный код (protoc-gen-go, -go-grpc, -connect-go)
internal/greeter/            — общая бизнес-логика
cmd/server-grpc/             — сервер на grpc-go            (порт :50051)
cmd/server-connect/          — сервер на connect-go          (порт :8080)
cmd/client-grpc/             — клиент на grpc-go
cmd/client-connect/          — клиент на connect-go (флаг -protocol)
```

## Требования

- Go 1.22+
- [buf](https://buf.build) и плагины (только для перегенерации кода):

```bash
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

## Кодогенерация

```bash
buf lint
buf generate
```

## Запуск

```bash
go run ./cmd/server-grpc      # классический gRPC на :50051
go run ./cmd/server-connect   # ConnectRPC на :8080 (Connect + gRPC + gRPC-Web)
```

Клиенты:

```bash
go run ./cmd/client-grpc                        # grpc-go клиент -> grpc-go сервер

# один connect-клиент, три протокола — все против одного сервера :8080
go run ./cmd/client-connect -protocol connect
go run ./cmd/client-connect -protocol grpc
go run ./cmd/client-connect -protocol grpcweb
```

## Киллер-фича: curl вместо grpcurl

Connect-сервер отвечает на обычный POST с JSON — никакой рефлексии
и специальных инструментов:

```bash
curl --header "Content-Type: application/json" \
  --data '{"name": "Хабр"}' \
  http://localhost:8080/greet.v1.GreetService/Greet
# {"greeting":"Привет, Хабр!"}
```

Health check живёт в том же HTTP-сервере:

```bash
curl http://localhost:8080/healthz
```
