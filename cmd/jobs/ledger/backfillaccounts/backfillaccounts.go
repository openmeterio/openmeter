package backfillaccounts

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	ledgerbackfillservice "github.com/openmeterio/openmeter/cmd/jobs/ledger/service"
	accountadapter "github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	accountservice "github.com/openmeterio/openmeter/openmeter/ledger/account/service"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	resolversadapter "github.com/openmeterio/openmeter/openmeter/ledger/resolvers/adapter"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

var (
	createdBefore    string
	customerPageSize int
	dryRun           bool
	continueOnError  bool
	includeDeleted   bool
)

var Cmd = &cobra.Command{
	Use:   "backfill-accounts",
	Short: "Backfill customer and business ledger accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := newService()
		if err != nil {
			return err
		}

		targetNamespace := internal.App.NamespaceManager.GetDefaultNamespace()

		var createdBeforeTime *time.Time
		if strings.TrimSpace(createdBefore) != "" {
			parsed, parseErr := time.Parse(time.RFC3339, createdBefore)
			if parseErr != nil {
				return fmt.Errorf("invalid created-before value %q: %w", createdBefore, parseErr)
			}

			parsed = parsed.UTC()
			createdBeforeTime = &parsed
		}

		output, err := service.Run(cmd.Context(), ledgerbackfillservice.RunInput{
			Namespace:               targetNamespace,
			DryRun:                  dryRun,
			ContinueOnError:         continueOnError,
			IncludeDeletedCustomers: includeDeleted,
			CustomerPageSize:        customerPageSize,
			CreatedBefore:           createdBeforeTime,
		})
		if err != nil {
			printSummary(output)
			return err
		}

		printSummary(output)

		if output.Result.FailureCount > 0 {
			return fmt.Errorf("backfill completed with %d failures", output.Result.FailureCount)
		}

		return nil
	},
}

func init() {
	Cmd.Flags().StringVar(&createdBefore, "created-before", "", "process only customers created before this RFC3339 timestamp")
	Cmd.Flags().IntVar(&customerPageSize, "customer-page-size", ledgerbackfillservice.DefaultCustomerPageSize, "number of customers to process per page")
	Cmd.Flags().BoolVar(&dryRun, "dry-run", false, "calculate what would be provisioned without writing")
	Cmd.Flags().BoolVar(&continueOnError, "continue-on-error", false, "continue after per-customer or per-namespace failures")
	Cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "include soft-deleted customers")
}

func newService() (*ledgerbackfillservice.Service, error) {
	locker, err := lockr.NewLocker(&lockr.LockerConfig{Logger: internal.App.Logger})
	if err != nil {
		return nil, fmt.Errorf("create locker: %w", err)
	}

	// We intentionally build the concrete resolver stack here because the public
	// wired account-resolver surface is narrowed and doesn't expose CreateCustomerAccounts.
	accountRepo := accountadapter.NewRepo(internal.App.EntClient)
	accountSvc := accountservice.New(accountRepo, locker)

	resolverRepo := resolversadapter.NewRepo(internal.App.EntClient)
	accountResolver := resolvers.NewAccountResolver(resolvers.AccountResolverConfig{
		AccountService: accountSvc,
		Repo:           resolverRepo,
		Locker:         locker,
	})

	service, err := ledgerbackfillservice.NewService(ledgerbackfillservice.Config{
		CustomerLister:     ledgerbackfillservice.NewEntCustomerLister(internal.App.EntClient),
		AccountProvisioner: accountResolver,
		Logger:             internal.App.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create backfill service: %w", err)
	}

	return service, nil
}

func printSummary(out ledgerbackfillservice.RunOutput) {
	ns := out.Result
	fmt.Printf(
		"namespace=%s business(already=%d would=%d provisioned=%d) customers(scanned=%d skipped_recent=%d already=%d would=%d provisioned=%d failures=%d)\n",
		ns.Namespace,
		ns.BusinessAlreadyProvisioned,
		ns.BusinessWouldProvision,
		ns.BusinessProvisioned,
		ns.CustomersScanned,
		ns.CustomersSkippedRecent,
		ns.CustomersAlreadyProvisioned,
		ns.CustomersWouldProvision,
		ns.CustomersProvisioned,
		ns.FailureCount,
	)
}
