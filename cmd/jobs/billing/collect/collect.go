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
		Short: "List gathering invoices which can be collected",
		RunE: func(cmd *cobra.Command, args []string) error {
			invoices, err := internal.App.BillingCollector.ListCollectableInvoices(cmd.Context(),
				billingworkercollect.ListCollectableInvoicesInput{
					Namespaces:   namespaces,
					InvoiceIDs:   invoiceIDs,
					Customers:    customerIDs,
					CollectionAt: time.Now(),
				})
			if err != nil {
				return err
			}

			for _, invoice := range invoices {
				fmt.Printf("Namespace: %s ID: %s CollectAt: %s\n", invoice.Namespace, invoice.ID, invoice.NextCollectionAt)
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
