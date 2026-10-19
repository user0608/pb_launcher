package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"pb_launcher/configs"
	"pb_launcher/internal/certificates/tlscommon"
	"pb_launcher/utils/domainutil"

	"go.uber.org/fx"
)

func RunHTTPSProxy(
	lc fx.Lifecycle,
	proxyHandler *DynamicReverseProxy,
	certStore tlscommon.Store,
	cfg configs.Config,
) {
	wildcardDomain := domainutil.ToWildcardDomain(cfg.GetDomain())

	addr := fmt.Sprintf("%s:%s", cfg.GetListenIPAddress(), cfg.GetHttpsPort())
	fmt.Println("https:addr:", addr)
	server := &http.Server{
		Addr: addr,
		TLSConfig: &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				domain := hello.ServerName
				if domainutil.SubdomainMatchesWildcard(domain, wildcardDomain) {
					domain = wildcardDomain
				}
				cert, err := certStore.Resolve(domain)
				if err != nil {
					slog.Error("resolve certificate", "domain", hello.ServerName, "error", err)
					return nil, err
				}
				tlsCert, err := tls.X509KeyPair(cert.CertPEM, cert.KeyPEM)
				if err != nil {
					slog.Error("parse certificate key pair", "domain", hello.ServerName, "error", err)
					return nil, err
				}
				return &tlsCert, err
			},
		},
		Handler: newProxyHandler(proxyHandler, cfg.GetDomain()),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				err := server.ListenAndServeTLS("", "")
				if err != nil && err != http.ErrServerClosed {
					slog.Error("HTTPS proxy encountered an error", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			slog.Info("shutting down HTTPS proxy", "addr", server.Addr)
			return server.Shutdown(ctx)
		},
	})
}
