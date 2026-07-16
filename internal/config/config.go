// Package config содержит вспомогательные функции для загрузки конфигурации
// приложений из JSON-файла: определение пути к файлу и разбор длительностей.
package config

import (
	"strings"
	"time"
)

// Path определяет путь к файлу конфигурации. Значение флага -c/-config в args
// имеет приоритет над envValue (переменной окружения CONFIG).
func Path(args []string, envValue string) string {
	path := envValue
	prefixes := []string{"-c=", "--c=", "-config=", "--config="}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-c" || arg == "--c" || arg == "-config" || arg == "--config" {
			if i+1 < len(args) {
				path = args[i+1]
				i++
			}
			continue
		}
		for _, p := range prefixes {
			if strings.HasPrefix(arg, p) {
				path = strings.TrimPrefix(arg, p)
			}
		}
	}
	return path
}

// Seconds преобразует строку длительности (например, "1s", "500ms") в целое
// число секунд.
func Seconds(d string) (int, error) {
	dur, err := time.ParseDuration(d)
	if err != nil {
		return 0, err
	}
	return int(dur.Seconds()), nil
}
