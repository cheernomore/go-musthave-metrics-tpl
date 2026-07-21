package netutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ipNet(t *testing.T, cidr string) net.Addr {
	t.Helper()
	ip, n, err := net.ParseCIDR(cidr)
	require.NoError(t, err)
	n.IP = ip
	return n
}

func TestFirstNonLoopbackIPv4(t *testing.T) {
	tests := []struct {
		name  string
		addrs []net.Addr
		want  string
	}{
		{
			name:  "пропускает loopback",
			addrs: []net.Addr{ipNet(t, "127.0.0.1/8"), ipNet(t, "192.168.1.10/24")},
			want:  "192.168.1.10",
		},
		{
			name:  "пропускает IPv6, берёт IPv4",
			addrs: []net.Addr{ipNet(t, "fe80::1/64"), ipNet(t, "10.0.0.5/8")},
			want:  "10.0.0.5",
		},
		{
			name:  "только loopback — nil",
			addrs: []net.Addr{ipNet(t, "127.0.0.1/8")},
			want:  "",
		},
		{
			name:  "пустой список — nil",
			addrs: nil,
			want:  "",
		},
		{
			name:  "не *net.IPNet игнорируется",
			addrs: []net.Addr{&net.TCPAddr{IP: net.ParseIP("8.8.8.8")}, ipNet(t, "172.16.0.3/12")},
			want:  "172.16.0.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FirstNonLoopbackIPv4(tt.addrs)
			if tt.want == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestLocalIP(t *testing.T) {
	ip, err := LocalIP()
	if err != nil {
		t.Skipf("IP-адрес хоста недоступен в этом окружении: %v", err)
	}

	require.NotNil(t, ip)
	assert.False(t, ip.IsUnspecified())
}
