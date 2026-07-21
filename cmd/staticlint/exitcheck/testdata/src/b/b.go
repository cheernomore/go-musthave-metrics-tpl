package b

import "os"

// Exit в пакете, отличном от main, не анализируется и не вызывает диагностики.
func Exit() {
	os.Exit(1)
}
