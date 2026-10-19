package systemstatus

import (
	"pb_launcher/configs"
	"pb_launcher/internal/launcher/domain"

	"go.uber.org/fx"
)

var Module = fx.Module("systemstatus",
	fx.Provide(func(cfg configs.Config, launcher *domain.LauncherManager) *Collector {
		return NewCollector(cfg.GetDataDir(), launcher.RunningProcessIDs)
	}),
	fx.Invoke(RegisterRoutes),
)
