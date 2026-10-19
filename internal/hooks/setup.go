package hooks

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"pb_launcher/configs"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterAdminExistsRoute(app *pocketbase.PocketBase, c configs.Config) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/proxy_configs", func(e *core.RequestEvent) error {
			response := map[string]any{
				"use_https":   c.IsHttpsEnabled(),
				"http_port":   c.GetHttpPort(),
				"https_port":  c.GetHttpsPort(),
				"base_domain": c.GetDomain(),
				"public_url":  c.GetPublicURL(),
			}
			return e.JSON(http.StatusOK, response)
		})

		se.Router.GET("/x-api/setup/admin-exists", func(e *core.RequestEvent) error {
			total, err := app.CountRecords(core.CollectionNameSuperusers, dbx.Not(dbx.HashExp{
				"email": core.DefaultInstallerEmail,
			}))
			if err != nil {
				return e.InternalServerError("failed to check admin existence", err)
			}
			if total == 0 {
				return e.JSON(http.StatusOK, map[string]any{"message": "no"})
			}
			return e.JSON(http.StatusOK, map[string]any{"message": "yes"})
		})

		se.Router.POST("/x-api/setup/admin", func(e *core.RequestEvent) error {
			var record struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := e.BindBody(&record); err != nil {
				return e.BadRequestError("invalid JSON body", err)
			}
			superusers, err := e.App.FindCachedCollectionByNameOrId(core.CollectionNameSuperusers)
			if err != nil {
				return err
			}
			users, err := e.App.FindCachedCollectionByNameOrId("users")
			if err != nil {
				return err
			}
			newSuperuser := core.NewRecord(superusers)
			newSuperuser.Set("email", record.Email)
			newSuperuser.Set("password", record.Password)

			newUser := core.NewRecord(users)
			newUser.Set("email", record.Email)
			newUser.Set("password", record.Password)
			newUser.Set("verified", true)

			txErr := e.App.RunInTransaction(func(txApp core.App) error {
				if err := txApp.Save(newSuperuser); err != nil {
					return err
				}
				existingUser, err := txApp.FindAuthRecordByEmail(users, record.Email)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("failed to find auth record: %w", err)
				}
				if existingUser == nil {
					if err := txApp.Save(newUser); err != nil {
						return fmt.Errorf("failed to save new user: %w", err)
					}
				}
				return nil
			})
			if txErr != nil {
				return e.InternalServerError("transaction failed", txErr)
			}
			return e.NoContent(http.StatusOK)
		})
		return se.Next()
	})
}
