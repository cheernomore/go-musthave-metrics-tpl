package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("start")
	os.Exit(1) // want "прямой вызов os.Exit в функции main запрещён"
}

// helper демонстрирует, что os.Exit вне функции main не считается нарушением.
func helper() {
	os.Exit(2)
}
