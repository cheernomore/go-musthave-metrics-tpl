// Package netutil содержит вспомогательные функции для работы с сетевыми
// адресами хоста.
package netutil

import (
	"errors"
	"net"
)

// ErrNoAddress возвращается, когда IP-адрес хоста определить не удалось.
var ErrNoAddress = errors.New("netutil: не удалось определить IP-адрес хоста")

// LocalIP определяет IP-адрес хоста. Сначала выясняется адрес исходящего
// интерфейса (UDP-«соединение» не отправляет данных), затем, если это не
// удалось, перебираются сетевые интерфейсы в поисках непетлевого IPv4.
func LocalIP() (net.IP, error) {
	if conn, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok &&
			addr.IP != nil && !addr.IP.IsUnspecified() {
			return addr.IP, nil
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	if ip := FirstNonLoopbackIPv4(addrs); ip != nil {
		return ip, nil
	}
	return nil, ErrNoAddress
}

// FirstNonLoopbackIPv4 возвращает первый непетлевой IPv4-адрес из списка
// или nil, если такого нет.
func FirstNonLoopbackIPv4(addrs []net.Addr) net.IP {
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return ip4
		}
	}
	return nil
}
