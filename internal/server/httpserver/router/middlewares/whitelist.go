package middlewares

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Whitelist returns a middleware that allows requests only from IPs in the trusted
// subnet. If the trusted subnet is not set, all IPs are allowed. The middleware
// checks the X-Real-IP and X-Forwarded-For headers in order, and if neither is set,
// it uses the remote address. If the IP is not in the trusted subnet, it returns a
// 403.
func (m *Middlewares) Whitelist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the trusted subnet is not set, allow all IPs.
		if m.trustedSubnet == nil {
			next.ServeHTTP(w, r)

			return
		}

		ip, err := getRemoteIPAddr(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// If the IP is not in the trusted subnet, return a 403.
		if !m.trustedSubnet.Contains(ip) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// getRemoteIPAddr returns the remote IP address of the request. It first tries to
// parse the X-Real-IP header, then the X-Forwarded-For header, and finally the
// remote address. If any of these fail, it returns an error.
func getRemoteIPAddr(req *http.Request) (net.IP, error) {
	// Try the X-Real-IP header.
	ip := net.ParseIP(req.Header.Get("X-Real-IP"))
	if ip == nil {
		// If the X-Real-IP header is not set, try the X-Forwarded-For header.
		ips := strings.Split(req.Header.Get("X-Forwarded-For"), ",")

		if len(ips) > 0 {
			// Try to parse the first IP in the list.
			ip = net.ParseIP(ips[0])
		}
	}

	// If the IP is still not set, use the remote address.
	if ip == nil {
		ipStr, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse remote address: %w", err)
		}

		ip = net.ParseIP(ipStr)
	}

	return ip, nil
}
