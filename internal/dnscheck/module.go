package dnscheck

import "go.uber.org/fx"

var Module = fx.Module("dnscheck",
	fx.Provide(NewChecker),
	fx.Invoke(RegisterRoutes),
)
