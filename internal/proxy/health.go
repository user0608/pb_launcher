package proxy

import (
	"net"
	"net/http"
	"strings"
)

// newProxyHandler exposes launcher liveness without intercepting health routes
// belonging to managed services or custom proxy targets. It does not probe them.
func newProxyHandler(next http.Handler, domain string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		ip := net.ParseIP(host)
		launcherHost := strings.EqualFold(host, domain) || strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
		if r.URL.Path != "/healthz" || !launcherHost {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("ok\n"))
		}
	})
}
