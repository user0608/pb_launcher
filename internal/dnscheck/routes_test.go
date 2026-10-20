package dnscheck

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestDNSCheckRoutes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, method, body  string
		authenticated, busy bool
		status              int
	}{
		{"anonymous", "POST", `{"domain":"app.test"}`, false, false, 401},
		{"valid", "POST", `{"domain":"app.test","expected_ips":["203.0.113.10"]}`, true, false, 200},
		{"invalid domain", "POST", `{"domain":"https://app.test"}`, true, false, 400},
		{"invalid IP", "POST", `{"domain":"app.test","expected_ips":["bad"]}`, true, false, 400},
		{"malformed JSON", "POST", `{`, true, false, 400},
		{"bounded payload", "POST", `{"domain":"` + strings.Repeat("a", 9000) + `"}`, true, false, 413},
		{"capacity limit", "POST", `{"domain":"app.test"}`, true, true, 429},
		{"GET not registered", "GET", "", true, false, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
			r := router.NewRouter(func(w http.ResponseWriter, req *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
				e := &core.RequestEvent{App: app}
				e.Response, e.Request = w, req
				if tc.authenticated {
					e.Auth = core.NewRecord(core.NewAuthCollection("users"))
				}
				return e, nil
			})
			checker := NewChecker()
			calls := 0
			checker.resolver = fakeResolver{ips: func(context.Context, string) ([]net.IPAddr, error) {
				calls++
				return []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}, nil
			}}
			if tc.busy {
				for range cap(checker.slots) {
					checker.slots <- struct{}{}
				}
			}
			RegisterRoutes(app, checker)
			if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: r}); err != nil {
				t.Fatal(err)
			}
			handler, err := r.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, "/x-api/dns/check", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if tc.status != 200 && calls != 0 {
				t.Fatal("rejected request reached resolver")
			}
			if tc.status == 200 {
				var result Result
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Status != "match" || response.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("result=%+v", result)
				}
			}
			if tc.busy && response.Header().Get("Retry-After") == "" {
				t.Fatal("missing Retry-After")
			}
		})
	}
}
