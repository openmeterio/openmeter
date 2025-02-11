package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/cmd/jobs/billing"
	"github.com/openmeterio/openmeter/cmd/jobs/entitlement"
	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	"github.com/openmeterio/openmeter/pkg/paniclogger"
)

var configFileName string

var rootCmd = cobra.Command{
	Use:           "jobs",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	defer paniclogger.PanicLogger()

	// Create os.Signal aware context.Context which will trigger context cancellation
	// upon receiving any of the listed signals.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	defer cancel()

	err := internal.InitializeApplication(ctx, configFileName)
	if err != nil {
		slog.Error("failed to initialize application", "error", err)

		// Call cleanup function is may not set yet
		if internal.AppShutdown != nil {
			internal.AppShutdown()
		}

		os.Exit(1)
	}
	defer internal.AppShutdown()

	if err = rootCmd.ExecuteContext(ctx); err != nil {
		slog.Error("failed to execute command", "error", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFileName, "config", "", "config.yaml", "config file (default is config.yaml)")
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(entitlement.RootCommand())
	rootCmd.AddCommand(billing.Cmd)
}
