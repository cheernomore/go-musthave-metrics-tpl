package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileObserver — приёмник аудита, дописывающий события в файл.
// Каждое событие добавляется в конец файла отдельной JSON-строкой.
type FileObserver struct {
	mu   sync.Mutex
	path string
}

// NewFileObserver создаёт файловый приёмник аудита для указанного пути.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Update сериализует событие в JSON и дописывает его новой строкой в файл.
func (o *FileObserver) Update(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	f, err := os.OpenFile(o.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}

	return nil
}
