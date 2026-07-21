package grpcapi

import (
	"context"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RealIPMetadataKey — ключ метаданных, в котором агент передаёт свой IP-адрес.
const RealIPMetadataKey = "x-real-ip"

// TrustedSubnetInterceptor создаёт unary-интерцептор, проверяющий, что IP-адрес
// агента из метаданных RealIPMetadataKey принадлежит доверенной подсети cidr.
// Если cidr пуст, ограничений нет. При непрохождении проверки возвращается
// ошибка с кодом codes.PermissionDenied.
func TrustedSubnetInterceptor(cidr string) (grpc.UnaryServerInterceptor, error) {
	if cidr == "" {
		return func(
			ctx context.Context,
			req any,
			_ *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (any, error) {
			return handler(ctx, req)
		}, nil
	}

	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("некорректная доверенная подсеть %q: %w", cidr, err)
	}

	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !trusted(ctx, subnet) {
			return nil, status.Error(codes.PermissionDenied, "agent address is not in trusted subnet")
		}
		return handler(ctx, req)
	}, nil
}

// trusted сообщает, принадлежит ли IP агента из метаданных подсети subnet.
func trusted(ctx context.Context, subnet *net.IPNet) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}

	values := md.Get(RealIPMetadataKey)
	if len(values) == 0 {
		return false
	}

	ip := net.ParseIP(strings.TrimSpace(values[0]))
	return ip != nil && subnet.Contains(ip)
}
