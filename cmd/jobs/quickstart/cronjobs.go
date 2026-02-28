package quickstart

import (
	"log/slog"
	"os"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	reconciler "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/reconciler"
)

var Cron = &cobra.Command{
	Use:   "cron",
	Short: "Schedule the required cron jobs in the background.",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := gocron.NewScheduler(
			gocron.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))),
		)
		if err != nil {
			return err
		}

		namespaces := []string{"default"}
		batchSize := 10

		// Sync subscriptions every hour
		_, err = s.NewJob(
			gocron.DurationJob(time.Hour),
			gocron.NewTask(func() {
				slog.Info("Syncing subscriptions")

				err := internal.App.BillingSubscriptionReconciler.All(cmd.Context(), reconciler.ReconcilerAllInput{
					ReconcilerListSubscriptionsInput: reconciler.ReconcilerListSubscriptionsInput{
						Namespaces: namespaces,
						Lookback:   time.Hour,
					},
					Force: false,
				})
				if err != nil {
					slog.Error("Error syncing subscriptions", "error", err)
				}
			}),
		)
		if err != nil {
			return err
		}

		// Collect invoices every minute
		_, err = s.NewJob(
			gocron.DurationJob(time.Minute),
			gocron.NewTask(func() {
				slog.Info("Collecting invoices")

				err := internal.App.BillingCollector.All(cmd.Context(), namespaces, nil, batchSize)
				if err != nil {
					slog.Error("Error collecting invoices", "error", err)
				}
			}),
		)
		if err != nil {
			return err
		}

		// Advance invoices every minute
		_, err = s.NewJob(
			gocron.DurationJob(time.Minute),
			gocron.NewTask(func() {
				slog.Info("Advancing invoices")

				err := internal.App.BillingAutoAdvancer.All(cmd.Context(), namespaces, batchSize)
				if err != nil {
					slog.Error("Error advancing invoices", "error", err)
				}
			}),
		)
		if err != nil {
			return err
		}

		// Sync LLM cost prices every 6 hours
		_, err = s.NewJob(
			gocron.DurationJob(6*time.Hour),
			gocron.NewTask(func() {
				slog.Info("Syncing LLM cost prices")

				err := internal.App.LLMCostSyncJob.Run(cmd.Context())
				if err != nil {
					slog.Error("Error syncing LLM cost prices", "error", err)
				}
			}),
		)
		if err != nil {
			return err
		}

		s.Start()

		<-cmd.Context().Done()
		if err := s.Shutdown(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	Cmd.AddCommand(Cron)
}
