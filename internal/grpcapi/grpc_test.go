package grpcapi

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	pb "github.com/cheernomore/go-musthave-metrics-tpl/internal/proto"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errSave = errors.New("сбой хранилища")

// failingRepo возвращает ошибку на сохранении батча.
type failingRepo struct {
	repository.MetricsRepository
}

func (failingRepo) SaveBatch([]models.Metrics) error { return errSave }

// startTestServer поднимает gRPC-сервер на случайном порту и возвращает адрес.
func startTestServer(t *testing.T, repo repository.MetricsRepository, cidr string) string {
	t.Helper()

	interceptor, err := TrustedSubnetInterceptor(cidr)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	pb.RegisterMetricsServer(srv, NewMetricsServer(repo))

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	return lis.Addr().String()
}

func sampleBatch() []models.Metrics {
	v := 42.5
	d := int64(3)
	return []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &v},
		{ID: "PollCount", MType: models.Counter, Delta: &d},
	}
}

func TestClientServer_RoundTrip(t *testing.T) {
	repo := repository.NewMemStorage()
	addr := startTestServer(t, repo, "")

	client, err := NewClient(addr, "")
	require.NoError(t, err)
	defer client.Close()

	require.NoError(t, client.Send(context.Background(), sampleBatch()))

	got, err := repo.Find("Alloc", models.Gauge)
	require.NoError(t, err)
	require.NotNil(t, got.Value)
	assert.InDelta(t, 42.5, *got.Value, 1e-9)

	counter, err := repo.Find("PollCount", models.Counter)
	require.NoError(t, err)
	require.NotNil(t, counter.Delta)
	assert.Equal(t, int64(3), *counter.Delta)
}

func TestClientServer_TrustedSubnetAllowed(t *testing.T) {
	repo := repository.NewMemStorage()
	addr := startTestServer(t, repo, "127.0.0.0/8")

	client, err := NewClient(addr, "127.0.0.1")
	require.NoError(t, err)
	defer client.Close()

	require.NoError(t, client.Send(context.Background(), sampleBatch()))

	all, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestClientServer_TrustedSubnetDenied(t *testing.T) {
	tests := []struct {
		name   string
		realIP string
	}{
		{"IP вне подсети", "10.1.2.3"},
		{"IP не передан", ""},
		{"некорректный IP", "not-an-ip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			addr := startTestServer(t, repo, "192.168.100.0/24")

			client, err := NewClient(addr, tt.realIP)
			require.NoError(t, err)
			defer client.Close()

			err = client.Send(context.Background(), sampleBatch())
			require.Error(t, err)
			assert.Equal(t, codes.PermissionDenied, status.Code(err))

			all, err := repo.FindAll()
			require.NoError(t, err)
			assert.Empty(t, all, "метрики не должны сохраняться")
		})
	}
}

func TestUpdateMetrics_EmptyRequest(t *testing.T) {
	srv := NewMetricsServer(repository.NewMemStorage())

	resp, err := srv.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestUpdateMetrics_RepoError(t *testing.T) {
	srv := NewMetricsServer(failingRepo{})

	_, err := srv.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{
		Metrics: ToProto(sampleBatch()),
	})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestTrustedSubnetInterceptor_InvalidCIDR(t *testing.T) {
	_, err := TrustedSubnetInterceptor("не CIDR")
	assert.Error(t, err)
}

func TestClient_SendUnreachable(t *testing.T) {
	// Сервер по адресу не слушает: отправка должна завершиться ошибкой.
	client, err := NewClient("127.0.0.1:1", "")
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	assert.Error(t, client.Send(ctx, sampleBatch()))
}
