package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"pb_launcher/configs"
	"pb_launcher/helpers/logstore"
	"pb_launcher/helpers/serialexecutor"
	"pb_launcher/helpers/unzip"
	"pb_launcher/helpers/zip"
	"pb_launcher/internal"
	"pb_launcher/internal/backup"
	"runtime"
	"strings"

	"pb_launcher/internal/certificates"
	"pb_launcher/internal/certmanager"
	"pb_launcher/internal/dnscheck"
	"pb_launcher/internal/download"
	"pb_launcher/internal/hookmanager"
	"pb_launcher/internal/launcher"
	"pb_launcher/internal/operationlog"
	"pb_launcher/internal/proxy"
	"pb_launcher/internal/repositorymanager"
	"pb_launcher/internal/systemstatus"
	_ "pb_launcher/migrations"

	_ "embed"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

//go:embed config.yaml.example
var configExampleYml []byte

//go:embed version
var version string

var commit = "none"

func initializeServer() *pocketbase.PocketBase {
	app := pocketbase.New()
	if err := app.Bootstrap(); err != nil {
		slog.Error("Failed to bootstrap PocketBase", "error", err)
		os.Exit(1)
	}
	return app
}

var skipCommands = map[string]bool{
	"gen-config": true,
	"version":    true,
}

func main() {

	args := strings.Join(os.Args[1:], "")
	skipInit := skipCommands[args]

	var app core.App
	if !skipInit {
		app = initializeServer()
	}
	rootCmd := createRootCommand(app)
	registerCommands(rootCmd, app)
	executeRootCommand(rootCmd)
}

func createRootCommand(app core.App) *cobra.Command {
	var configFile string
	comand := &cobra.Command{
		Use: path.Base(os.Args[0]),
		Run: func(cmd *cobra.Command, args []string) {
			fx.New(
				fx.Provide(func() (configs.Config, error) {
					return configs.LoadConfigs(configFile)
				}),
				certificates.Module,
				fx.Provide(configs.NewPBServeConfig),
				fx.Provide(unzip.NewUnzip),
				fx.Provide(zip.NewZip),
				fx.Provide(logstore.NewServiceLogDB),
				fx.Provide(operationlog.New),
				fx.Provide(serialexecutor.NewSequentialExecutor),
				fx.Supply(app),
				download.Module,
				launcher.Module,
				proxy.Module,
				certmanager.Module,
				backup.Module,
				hookmanager.Module,
				repositorymanager.Module,
				systemstatus.Module,
				dnscheck.Module,
				internal.Module, // hooks
				fx.Invoke(
					StartApiServer,
					ServeEmbeddedUI,
					// Tasks
					RegisterCertificateAutoRenewal,
					RegisterCertRequestPlanner,
					RegisterCertRequestExecutor,

					RegisterBinaryReleaseSync,
					RegisterLauncherRunner,
					RunSequentialExecutor, // Start Stask Runner
				),
			).Run()
		},
	}
	comand.Flags().StringVarP(&configFile, "config", "c", "", "Path to the config file (yml)")
	return comand
}

func buildUpgradeCommand(migrationsRunner *core.MigrationsRunner) *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the database schema to the latest version",
		Run: func(cmd *cobra.Command, args []string) {
			applied, err := migrationsRunner.Up()
			if err != nil {
				slog.Error("Database upgrade failed", "error", err)
				os.Exit(1)
			}
			if len(applied) == 0 {
				slog.Info("No new migrations to apply")
				return
			}
			for _, file := range applied {
				slog.Info("Migration applied", "file", file)
			}
		},
	}
}

func buildDowngradeCommand(migrationsRunner *core.MigrationsRunner) *cobra.Command {
	return &cobra.Command{
		Use:   "downgrade",
		Short: "Downgrade the database schema to a previous version",
		Run: func(cmd *cobra.Command, args []string) {
			reverted, err := migrationsRunner.Down(1)
			if err != nil {
				slog.Error("Database downgrade failed", "error", err)
				os.Exit(1)
			}
			if len(reverted) == 0 {
				slog.Info("No migration to revert")
				return
			}
			for _, file := range reverted {
				slog.Debug("Reverted migration", "file", file)
			}
		},
	}
}

func buildGenConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gen-config",
		Short: "Generate the default configuration in YAML format",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(string(configExampleYml))
		},
	}
}

func buildVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print application version and environment details",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Version:", version)
			fmt.Println("Commit:", commit)
			fmt.Println("Go Version:", runtime.Version())
			fmt.Println("Compiler:", runtime.Compiler)
			fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}

func registerCommands(rootCmd *cobra.Command, app core.App) {
	migrationsRunner := core.NewMigrationsRunner(app, core.AppMigrations)
	rootCmd.AddCommand(buildUpgradeCommand(migrationsRunner))
	rootCmd.AddCommand(buildDowngradeCommand(migrationsRunner))
	rootCmd.AddCommand(buildGenConfigCommand())
	rootCmd.AddCommand(buildVersionCommand())
}

func executeRootCommand(rootCmd *cobra.Command) {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}
