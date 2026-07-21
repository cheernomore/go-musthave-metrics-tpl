package proto

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// stubServer — минимальная реализация сервиса для проверки сгенерированной
// обвязки регистрации и вызова.
type stubServer struct {
	UnimplementedMetricsServer

	got *UpdateMetricsRequest
}

func (s *stubServer) UpdateMetrics(
	_ context.Context,
	req *UpdateMetricsRequest,
) (*UpdateMetricsResponse, error) {
	s.got = req
	return &UpdateMetricsResponse{}, nil
}

func TestGeneratedService_RegisterAndCall(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	stub := &stubServer{}
	srv := grpc.NewServer()
	RegisterMetricsServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := NewMetricsClient(conn)
	resp, err := client.UpdateMetrics(context.Background(), &UpdateMetricsRequest{
		Metrics: []*Metric{{Id: "Alloc", Type: Metric_GAUGE, Value: 1}},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, stub.got)
	require.Len(t, stub.got.GetMetrics(), 1)
	assert.Equal(t, "Alloc", stub.got.GetMetrics()[0].GetId())
}

func TestUnimplementedServer_ReturnsError(t *testing.T) {
	var s UnimplementedMetricsServer
	_, err := s.UpdateMetrics(context.Background(), &UpdateMetricsRequest{})
	assert.Error(t, err)
}

func TestUpdateMetricsRequest_RoundTrip(t *testing.T) {
	req := &UpdateMetricsRequest{
		Metrics: []*Metric{
			{Id: "Alloc", Type: Metric_GAUGE, Value: 1.5},
			{Id: "PollCount", Type: Metric_COUNTER, Delta: 7},
		},
	}

	data, err := proto.Marshal(req)
	require.NoError(t, err)

	var got UpdateMetricsRequest
	require.NoError(t, proto.Unmarshal(data, &got))

	require.Len(t, got.GetMetrics(), 2)
	assert.Equal(t, "Alloc", got.GetMetrics()[0].GetId())
	assert.Equal(t, Metric_GAUGE, got.GetMetrics()[0].GetType())
	assert.InDelta(t, 1.5, got.GetMetrics()[0].GetValue(), 1e-9)

	assert.Equal(t, "PollCount", got.GetMetrics()[1].GetId())
	assert.Equal(t, Metric_COUNTER, got.GetMetrics()[1].GetType())
	assert.Equal(t, int64(7), got.GetMetrics()[1].GetDelta())
}

func TestUpdateMetricsResponse_RoundTrip(t *testing.T) {
	data, err := proto.Marshal(&UpdateMetricsResponse{})
	require.NoError(t, err)

	var got UpdateMetricsResponse
	require.NoError(t, proto.Unmarshal(data, &got))
}

func TestGetters_NilSafe(t *testing.T) {
	var m *Metric
	assert.Empty(t, m.GetId())
	assert.Equal(t, Metric_GAUGE, m.GetType())
	assert.Zero(t, m.GetDelta())
	assert.Zero(t, m.GetValue())

	var req *UpdateMetricsRequest
	assert.Nil(t, req.GetMetrics())
}

func TestMessages_ProtoMessageAPI(t *testing.T) {
	m := &Metric{Id: "Alloc", Type: Metric_GAUGE, Value: 2.5}
	assert.NotEmpty(t, m.String())
	assert.NotNil(t, m.ProtoReflect())
	m.Reset()
	assert.Empty(t, m.GetId())

	req := &UpdateMetricsRequest{Metrics: []*Metric{{Id: "X"}}}
	assert.NotEmpty(t, req.String())
	assert.NotNil(t, req.ProtoReflect())
	req.Reset()
	assert.Empty(t, req.GetMetrics())

	resp := &UpdateMetricsResponse{}
	assert.NotNil(t, resp.ProtoReflect())
	_ = resp.String()
	resp.Reset()
}

func TestMetricMType_Enum(t *testing.T) {
	assert.Equal(t, "GAUGE", Metric_GAUGE.String())
	assert.Equal(t, "COUNTER", Metric_COUNTER.String())

	assert.Equal(t, Metric_COUNTER, *Metric_COUNTER.Enum())
	assert.Equal(t, int32(0), int32(Metric_GAUGE.Number()))
	assert.Equal(t, int32(1), int32(Metric_COUNTER.Number()))

	assert.NotNil(t, Metric_GAUGE.Descriptor())
	assert.NotNil(t, Metric_GAUGE.Type())
}
