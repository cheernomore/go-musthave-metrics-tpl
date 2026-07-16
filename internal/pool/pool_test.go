package pool_test

import (
	"testing"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	"github.com/cheernomore/go-musthave-metrics-tpl/internal/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buffer — тестовый тип с состоянием и методом Reset().
type buffer struct {
	data   []int
	resets int
}

func (b *buffer) Reset() {
	b.data = b.data[:0]
	b.resets++
}

func TestPool_GetCreatesObject(t *testing.T) {
	created := 0
	p := pool.New(func() *buffer {
		created++
		return &buffer{}
	})

	obj := p.Get()
	require.NotNil(t, obj)
	assert.Equal(t, 1, created)
}

func TestPool_PutResets(t *testing.T) {
	p := pool.New(func() *buffer { return &buffer{} })

	obj := p.Get()
	obj.data = append(obj.data, 1, 2, 3)

	p.Put(obj)

	// Put должен был вызвать Reset перед возвратом в пул.
	assert.Empty(t, obj.data)
	assert.Equal(t, 1, obj.resets)
}

func TestPool_Reuse(t *testing.T) {
	created := 0
	p := pool.New(func() *buffer {
		created++
		return &buffer{}
	})

	first := p.Get()
	p.Put(first)
	second := p.Get()

	// После Put объект переиспользуется, новый экземпляр не создаётся.
	assert.Same(t, first, second)
	assert.Equal(t, 1, created)
}

// TestPool_WithGeneratedReset демонстрирует работу пула с *models.Metrics,
// для которого метод Reset() сгенерирован утилитой cmd/reset.
func TestPool_WithGeneratedReset(t *testing.T) {
	p := pool.New(func() *models.Metrics { return &models.Metrics{} })

	m := p.Get()
	m.ID = "Alloc"
	m.MType = models.Gauge
	v := 42.0
	m.Value = &v

	p.Put(m)

	assert.Empty(t, m.ID)
	assert.Empty(t, m.MType)
	require.NotNil(t, m.Value)
	assert.Equal(t, 0.0, *m.Value)
}
