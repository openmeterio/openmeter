package advance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

var namespace string

var Cmd = &cobra.Command{
	Use:   "advance",
	Short: "Invoice advance operations",
}

func init() {
	Cmd.AddCommand(ListCmd())
	Cmd.AddCommand(InvoiceCmd())
	Cmd.AddCommand(AllCmd())

	Cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace the operation should be performed")
}

var ListCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List invoices which can be advanced",
		RunE: func(cmd *cobra.Command, args []string) error {
			var ns []string
			if namespace != "" {
				ns = append(ns, namespace)
			}

			invoices, err := internal.App.BillingAutoAdvancer.ListInvoicesToAdvance(cmd.Context(), ns, nil)
			if err != nil {
				return err
			}

			for _, invoice := range invoices {
				fmt.Printf("Namespace: %s ID: %s DraftUntil: %s\n", invoice.Namespace, invoice.ID, invoice.DraftUntil)
			}

			return nil
		},
	}

	return cmd
}

var InvoiceCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoice [INVOICE_ID]",
		Short: "Advance invoice(s)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if namespace == "" {
				return fmt.Errorf("invoice namespace is required")
			}

			for _, invoiceID := range args {
				_, err := internal.App.BillingAutoAdvancer.AdvanceInvoice(cmd.Context(), billing.InvoiceID{
					Namespace: namespace,
					ID:        invoiceID,
				})
				if err != nil {
					return fmt.Errorf("failed to advance invoice %s: %w", invoiceID, err)
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
			var ns []string
			if namespace != "" {
				ns = append(ns, namespace)
			}

			return internal.App.BillingAutoAdvancer.All(cmd.Context(), ns, batchSize)
		},
	}

	cmd.PersistentFlags().IntVar(&batchSize, "batch", 0, "operation batch size")

	return cmd
}
