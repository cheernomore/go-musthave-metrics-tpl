package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// realIPHeader — заголовок, в котором агент передаёт свой IP-адрес.
const realIPHeader = "X-Real-IP"

// TrustedSubnetMiddleware создаёт middleware, ограничивающее приём метрик
// доверенной подсетью. Если cidr пуст, возвращается «прозрачное» middleware и
// метрики принимаются без ограничений. Иначе IP-адрес из заголовка X-Real-IP
// должен входить в подсеть, иначе возвращается 403 Forbidden.
func TrustedSubnetMiddleware(cidr string) (func(http.Handler) http.Handler, error) {
	if cidr == "" {
		return func(next http.Handler) http.Handler { return next }, nil
	}

	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("некорректная доверенная подсеть %q: %w", cidr, err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := net.ParseIP(strings.TrimSpace(r.Header.Get(realIPHeader)))
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, "agent address is not in trusted subnet", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}
