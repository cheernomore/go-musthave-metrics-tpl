package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPObserver — приёмник аудита, отправляющий события методом POST
// на заданный удалённый URL.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создаёт удалённый приёмник аудита для указанного URL.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Update сериализует событие в JSON и отправляет его POST-запросом на URL.
func (o *HTTPObserver) Update(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, o.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("send audit event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("audit endpoint returned status %d", resp.StatusCode)
	}

	return nil
}
