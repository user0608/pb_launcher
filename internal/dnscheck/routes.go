package dnscheck

import (
	"context"
	"errors"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRoutes(app *pocketbase.PocketBase, checker *Checker) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/x-api/dns/check", func(e *core.RequestEvent) error {
			var request Request
			if err := e.BindBody(&request); err != nil {
				return e.BadRequestError("invalid DNS check request", nil)
			}
			result, err := checker.Check(e.Request.Context(), request)
			switch {
			case errors.Is(err, ErrInvalidDomain), errors.Is(err, ErrInvalidIPs):
				return e.BadRequestError(err.Error(), nil)
			case errors.Is(err, ErrBusy):
				e.Response.Header().Set("Retry-After", "2")
				return e.TooManyRequestsError(err.Error(), nil)
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				return e.Error(http.StatusRequestTimeout, "DNS check canceled", nil)
			case err != nil:
				return e.InternalServerError("failed to check DNS", nil)
			}
			e.Response.Header().Set("Cache-Control", "no-store")
			return e.JSON(http.StatusOK, result)
		}).Bind(apis.RequireAuth(), apis.BodyLimit(8*1024))
		return se.Next()
	})
}
