package collect

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var (
	namespaces  []string
	customerIDs []string
	invoiceIDs  []string
)

var Cmd = &cobra.Command{
	Use:   "collect",
	Short: "Invoice collection operations",
}

func init() {
	Cmd.AddCommand(ListCmd())
	Cmd.AddCommand(InvoiceCmd())
	Cmd.AddCommand(AllCmd())
}

var ListCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List customers with gathering invoices ready for collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			customers, err := internal.App.BillingCollector.ListCustomersToCollect(cmd.Context(),
				billingworkercollect.ListCustomersToCollectInput{
					Namespaces:  namespaces,
					InvoiceIDs:  invoiceIDs,
					CustomerIDs: customerIDs,
					AsOf:        time.Now(),
				})
			if err != nil {
				return err
			}

			for _, customer := range customers {
				fmt.Printf("Namespace: %s Customer: %s\n", customer.Namespace, customer.ID)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")
	cmd.PersistentFlags().StringSliceVar(&customerIDs, "c", nil, "filter by customer ids")
	cmd.PersistentFlags().StringSliceVar(&invoiceIDs, "i", nil, "filter by invoice ids")

	return cmd
}

var InvoiceCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoice [CUSTOMER_ID]",
		Short: "Create new invoice(s) for customer(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, customerID := range args {
				_, err := internal.App.BillingCollector.CollectCustomerInvoice(cmd.Context(),
					billingworkercollect.CollectCustomerInvoiceInput{
						CustomerID: customer.CustomerID{
							Namespace: "default",
							ID:        customerID,
						},
						AsOf: time.Now(),
					},
				)
				if err != nil {
					return fmt.Errorf("failed to invoice customer %s: %w", customerID, err)
				}
			}

			return nil
		},
	}

	return cmd
}

var batchSize int

var AllCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Advance all eligible invoices",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.App.BillingCollector.All(cmd.Context(), namespaces, customerIDs, batchSize)
		},
	}

	cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")
	cmd.PersistentFlags().StringSliceVar(&customerIDs, "c", nil, "filter by customer ids")
	cmd.PersistentFlags().IntVar(&batchSize, "batch", 0, "operation batch size")

	return cmd
}
