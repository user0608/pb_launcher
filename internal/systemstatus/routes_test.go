package systemstatus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestStatusRouteAuthentication(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		authenticated bool
		status        int
	}{
		{"anonymous denied", false, http.StatusUnauthorized},
		{"authenticated allowed", true, http.StatusOK},
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
			collector := NewCollector("unused", func() map[string]int { return nil })
			calls := 0
			collector.collect = func(string, map[string]int) sample { calls++; return sample{} }
			RegisterRoutes(app, collector)
			if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: r}); err != nil {
				t.Fatal(err)
			}
			handler, err := r.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("GET", "/x-api/system/status", nil))
			if response.Code != tc.status {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			if !tc.authenticated && calls != 0 {
				t.Fatal("anonymous request collected metrics")
			}
			if tc.authenticated && response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing no-store header")
			}
		})
	}
}
