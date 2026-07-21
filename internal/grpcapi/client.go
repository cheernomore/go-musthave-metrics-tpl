package grpcapi

import (
	"context"

	models "github.com/cheernomore/go-musthave-metrics-tpl/internal/model"
	pb "github.com/cheernomore/go-musthave-metrics-tpl/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client — gRPC-клиент агента для отправки пакетов метрик на сервер.
type Client struct {
	conn   *grpc.ClientConn
	client pb.MetricsClient
	realIP string
}

// NewClient создаёт клиент, подключённый к target. Значение realIP передаётся
// серверу в метаданных запроса под ключом RealIPMetadataKey; пустое значение
// не добавляется.
func NewClient(target, realIP string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		client: pb.NewMetricsClient(conn),
		realIP: realIP,
	}, nil
}

// Send отправляет пакет метрик методом UpdateMetrics.
func (c *Client) Send(ctx context.Context, metrics []models.Metrics) error {
	if c.realIP != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, RealIPMetadataKey, c.realIP)
	}

	_, err := c.client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{
		Metrics: ToProto(metrics),
	})
	return err
}

// Close закрывает соединение с сервером.
func (c *Client) Close() error {
	return c.conn.Close()
}
