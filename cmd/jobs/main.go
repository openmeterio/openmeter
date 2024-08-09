package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/cmd/jobs/config"
	"github.com/openmeterio/openmeter/cmd/jobs/entitlement"
	"github.com/openmeterio/openmeter/cmd/jobs/service"
)

const (
	otelName = "openmeter.io/jobs"
)

func main() {
	var telemetry *service.Telemetry

	defer func() {
		if telemetry != nil && telemetry.Shutdown != nil {
			telemetry.Shutdown()
		}
	}()

	rootCmd := cobra.Command{
		Use: "jobs",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.LoadConfig(cmd.Flag("config").Value.String())
			if err != nil {
				return err
			}

			config.SetConfig(conf)

			telemetry, err = service.NewTelemetry(cmd.Context(), conf.Telemetry, conf.Environment, version, otelName)
			return err
		},
	}

	var configFileName string

	rootCmd.PersistentFlags().StringVarP(&configFileName, "config", "", "config.yaml", "config file (default is config.yaml)")
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(entitlement.RootCommand())

	if err := rootCmd.Execute(); err != nil {
		slog.Default().Error("failed to execute command", "error", err)
		os.Exit(1)
	}
}
