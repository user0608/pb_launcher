package systemstatus

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRoutes(app *pocketbase.PocketBase, collector *Collector) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/system/status", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Cache-Control", "no-store")
			return e.JSON(http.StatusOK, collector.Snapshot())
		}).Bind(apis.RequireAuth())
		return se.Next()
	})
}
