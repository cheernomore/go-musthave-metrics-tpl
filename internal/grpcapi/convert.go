// Package grpcapi содержит gRPC-реализацию обмена метриками между агентом и
// сервером: сервис Metrics, интерцептор проверки доверенной подсети и клиент.
package grpcapi

import (
	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	pb "github.com/cheernomore/go-musthave-metrics-tpl/internal/proto"
)

// ToProto преобразует доменные метрики в protobuf-представление.
func ToProto(metrics []models.Metrics) []*pb.Metric {
	out := make([]*pb.Metric, 0, len(metrics))
	for _, m := range metrics {
		item := &pb.Metric{Id: m.ID}
		switch m.MType {
		case models.Counter:
			item.Type = pb.Metric_COUNTER
			if m.Delta != nil {
				item.Delta = *m.Delta
			}
		case models.Gauge:
			item.Type = pb.Metric_GAUGE
			if m.Value != nil {
				item.Value = *m.Value
			}
		default:
			// Неизвестный тип метрики не отправляем.
			continue
		}
		out = append(out, item)
	}
	return out
}

// FromProto преобразует protobuf-метрики в доменную модель.
func FromProto(metrics []*pb.Metric) []models.Metrics {
	out := make([]models.Metrics, 0, len(metrics))
	for _, m := range metrics {
		if m == nil {
			continue
		}
		switch m.GetType() {
		case pb.Metric_COUNTER:
			delta := m.GetDelta()
			out = append(out, models.Metrics{
				ID:    m.GetId(),
				MType: models.Counter,
				Delta: &delta,
			})
		case pb.Metric_GAUGE:
			value := m.GetValue()
			out = append(out, models.Metrics{
				ID:    m.GetId(),
				MType: models.Gauge,
				Value: &value,
			})
		}
	}
	return out
}
