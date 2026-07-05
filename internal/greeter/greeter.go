// Package greeter содержит бизнес-логику, общую для gRPC- и Connect-серверов.
// Логика вынесена отдельно, чтобы сравнение фреймворков было честным:
// оба сервера — лишь транспортные обёртки вокруг одних и тех же функций.
package greeter

import "fmt"

// Greet формирует приветствие для unary-вызова.
func Greet(name string) string {
	return fmt.Sprintf("Привет, %s!", name)
}

// GreetSeq формирует приветствие для i-го сообщения стрима.
func GreetSeq(name string, i int32) string {
	return fmt.Sprintf("Привет, %s! (сообщение #%d)", name, i)
}
