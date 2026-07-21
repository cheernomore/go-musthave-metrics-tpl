package grpcapi

import (
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	pb "github.com/cheernomore/go-musthave-metrics-tpl/internal/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToProto(t *testing.T) {
	v := 12.5
	d := int64(7)
	in := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &v},
		{ID: "PollCount", MType: models.Counter, Delta: &d},
		{ID: "Broken", MType: "unknown"}, // неизвестный тип пропускается
		{ID: "NilGauge", MType: models.Gauge},
	}

	out := ToProto(in)

	require.Len(t, out, 3)
	assert.Equal(t, "Alloc", out[0].GetId())
	assert.Equal(t, pb.Metric_GAUGE, out[0].GetType())
	assert.InDelta(t, 12.5, out[0].GetValue(), 1e-9)

	assert.Equal(t, pb.Metric_COUNTER, out[1].GetType())
	assert.Equal(t, int64(7), out[1].GetDelta())

	// Метрика без значения передаётся с нулём.
	assert.Equal(t, "NilGauge", out[2].GetId())
	assert.InDelta(t, 0.0, out[2].GetValue(), 1e-9)
}

func TestFromProto(t *testing.T) {
	in := []*pb.Metric{
		{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 3.5},
		{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 9},
		nil, // пропускается
	}

	out := FromProto(in)

	require.Len(t, out, 2)
	assert.Equal(t, "Alloc", out[0].ID)
	assert.Equal(t, models.Gauge, out[0].MType)
	require.NotNil(t, out[0].Value)
	assert.InDelta(t, 3.5, *out[0].Value, 1e-9)
	assert.Nil(t, out[0].Delta)

	assert.Equal(t, models.Counter, out[1].MType)
	require.NotNil(t, out[1].Delta)
	assert.Equal(t, int64(9), *out[1].Delta)
	assert.Nil(t, out[1].Value)
}

func TestConvert_RoundTrip(t *testing.T) {
	v := 1.25
	d := int64(3)
	in := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &v},
		{ID: "PollCount", MType: models.Counter, Delta: &d},
	}

	out := FromProto(ToProto(in))

	require.Len(t, out, 2)
	assert.Equal(t, in[0].ID, out[0].ID)
	assert.InDelta(t, *in[0].Value, *out[0].Value, 1e-9)
	assert.Equal(t, *in[1].Delta, *out[1].Delta)
}

func TestConvert_Empty(t *testing.T) {
	assert.Empty(t, ToProto(nil))
	assert.Empty(t, FromProto(nil))
}
