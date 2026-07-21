package grpcapi

import (
	"context"

	pb "github.com/cheernomore/go-musthave-metrics-tpl/internal/proto"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsServer реализует gRPC-сервис Metrics.
type MetricsServer struct {
	pb.UnimplementedMetricsServer

	repo repository.MetricsRepository
}

// NewMetricsServer создаёт сервис поверх хранилища метрик.
func NewMetricsServer(repo repository.MetricsRepository) *MetricsServer {
	return &MetricsServer{repo: repo}
}

// UpdateMetrics принимает батч метрик и сохраняет его в хранилище. Метод
// подходит как для одиночных метрик, так и для пакетов.
func (s *MetricsServer) UpdateMetrics(
	_ context.Context,
	req *pb.UpdateMetricsRequest,
) (*pb.UpdateMetricsResponse, error) {
	metrics := FromProto(req.GetMetrics())
	if len(metrics) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	if err := s.repo.SaveBatch(metrics); err != nil {
		return nil, status.Error(codes.Internal, "не удалось сохранить метрики")
	}

	return &pb.UpdateMetricsResponse{}, nil
}
