package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyHealth(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, host, method, path string
		status                   int
		forwarded                bool
	}{
		{"base domain", "pb.test:7080", "GET", "/healthz", 200, false},
		{"loopback", "127.0.0.1:7080", "GET", "/healthz", 200, false},
		{"ipv6 loopback", "[::1]:7080", "HEAD", "/healthz", 200, false},
		{"method rejected", "localhost", "POST", "/healthz", 405, false},
		{"service route preserved", "service.pb.test", "GET", "/healthz", 202, true},
		{"custom domain preserved", "app.test", "GET", "/healthz", 202, true},
		{"other routes preserved", "pb.test", "GET", "/api/health", 202, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			forwarded := false
			h := newProxyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				forwarded = true
				w.WriteHeader(http.StatusAccepted)
			}), "pb.test")
			r := httptest.NewRequest(tc.method, "http://"+tc.host+tc.path, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || forwarded != tc.forwarded {
				t.Fatalf("status=%d forwarded=%v, want %d %v", w.Code, forwarded, tc.status, tc.forwarded)
			}
			if tc.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
		})
	}
}
