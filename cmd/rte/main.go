package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	rteapp "github.com/openmeterio/openmeter/app/rte"
)

func main() {
	if err := runService(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runService(ctx context.Context, args []string) error {
	v := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook()))
	flags := pflag.NewFlagSet("RTE", pflag.ContinueOnError)
	config.SetViperDefaults(v, flags)
	flags.String("config", "", "Configuration file")
	flags.Bool("version", false, "Show version information")
	flags.Bool("validate", false, "Validate configuration and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if showVersion, _ := flags.GetBool("version"); showVersion {
		fmt.Printf("OpenMeter RTE version %s\n", version)

		return nil
	}

	if path, _ := flags.GetString("config"); path != "" {
		v.SetConfigFile(path)
	}

	if err := v.ReadInConfig(); err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		return err
	}

	var conf config.Configuration
	if err := v.Unmarshal(&conf); err != nil {
		return err
	}

	if !conf.RTE.Enabled {
		return nil
	}

	if err := conf.Validate(); err != nil {
		return err
	}

	if !conf.Credits.Enabled {
		return errors.New("RTE requires credits.enabled")
	}

	if err := conf.Events.Validate(); err != nil {
		return fmt.Errorf("RTE events: %w", err)
	}

	if validate, _ := flags.GetBool("validate"); validate {
		return nil
	}

	app, cleanup, err := rteapp.Initialize(ctx, conf, common.NewMetadata(conf, version, "rte"))
	if cleanup != nil {
		defer cleanup()
	}

	if err != nil {
		return err
	}

	app.SetGlobals()
	if err := app.Migrate(ctx); err != nil {
		return err
	}

	app.Run()

	return nil
}
