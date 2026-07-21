package main

// Локально объявленный os не является пакетом os, поэтому вызов os.Exit
// здесь не должен приводить к диагностике.
func main() {
	os := struct {
		Exit func(int)
	}{
		Exit: func(int) {},
	}
	os.Exit(0)
}
