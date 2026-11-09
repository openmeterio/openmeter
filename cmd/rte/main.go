package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/app/config"
)

func main() {
	if err := runService(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runService(ctx context.Context) error {
	v := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook()))
	flags := pflag.NewFlagSet("RTE", pflag.ContinueOnError)
	config.SetViperDefaults(v, flags)
	flags.String("config", "", "Configuration file")
	flags.Bool("version", false, "Show version information")
	flags.Bool("validate", false, "Validate configuration and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
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

	if err := conf.Validate(); err != nil {
		return err
	}

	if !conf.Credits.Enabled {
		return errors.New("RTE requires credits.enabled")
	}

	if validate, _ := flags.GetBool("validate"); validate {
		return nil
	}

	app, cleanup, err := initializeApplication(ctx, conf)
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
