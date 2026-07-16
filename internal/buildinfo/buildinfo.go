// Package buildinfo выводит информацию о сборке приложения (версию, дату и
// коммит), которая обычно задаётся через -ldflags при сборке.
package buildinfo

import (
	"fmt"
	"io"
)

// Print печатает в w версию, дату и коммит сборки. Пустые значения
// заменяются на "N/A".
func Print(w io.Writer, version, date, commit string) {
	fmt.Fprintf(w, "Build version: %s\n", orNA(version))
	fmt.Fprintf(w, "Build date: %s\n", orNA(date))
	fmt.Fprintf(w, "Build commit: %s\n", orNA(commit))
}

// orNA возвращает s или "N/A", если значение пустое.
func orNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
