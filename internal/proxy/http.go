package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"pb_launcher/configs"

	"go.uber.org/fx"
)

func RunHttpProxy(lc fx.Lifecycle, handler *DynamicReverseProxy, cfg configs.Config) {
	addr := fmt.Sprintf("%s:%s", cfg.GetListenIPAddress(), cfg.GetHttpPort())
	fmt.Println("http:addr:", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: newProxyHandler(handler, cfg.GetDomain()),
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					slog.Error("proxy server error", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

}
