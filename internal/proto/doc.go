// Package proto содержит сгенерированный из metrics.proto код gRPC-контракта
// обмена метриками между агентом и сервером: сообщения Metric,
// UpdateMetricsRequest, UpdateMetricsResponse и сервис Metrics.
//
// Код генерируется командой:
//
//	protoc --go_out=. --go_opt=paths=source_relative \
//	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
//	       internal/proto/metrics.proto
//
// Файлы metrics.pb.go и metrics_grpc.pb.go редактировать вручную не следует.
package proto

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative metrics.proto
