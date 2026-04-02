package advancecharges

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/cmd/jobs/internal"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var namespace string

var Cmd = &cobra.Command{
	Use:   "advance-charges",
	Short: "Charge advance operations",
}

func init() {
	Cmd.AddCommand(ListCmd())
	Cmd.AddCommand(CustomerCmd())
	Cmd.AddCommand(AllCmd())

	Cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace the operation should be performed")
}

var ListCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List customers which have charges to advance",
		RunE: func(cmd *cobra.Command, args []string) error {
			if internal.App.ChargesAutoAdvancer == nil {
				return fmt.Errorf("charges are not enabled")
			}

			var ns []string
			if namespace != "" {
				ns = append(ns, namespace)
			}

			customers, err := internal.App.ChargesAutoAdvancer.ListCustomersToAdvance(cmd.Context(), ns)
			if err != nil {
				return err
			}

			for _, c := range customers {
				fmt.Printf("Namespace: %s CustomerID: %s\n", c.Namespace, c.ID)
			}

			return nil
		},
	}

	return cmd
}

var CustomerCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "customer [CUSTOMER_ID]",
		Short: "Advance charges for a customer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if internal.App.ChargesAutoAdvancer == nil {
				return fmt.Errorf("charges are not enabled")
			}

			if namespace == "" {
				return fmt.Errorf("namespace is required")
			}

			return internal.App.ChargesAutoAdvancer.AdvanceCharges(cmd.Context(), customer.CustomerID{
				Namespace: namespace,
				ID:        args[0],
			})
		},
	}

	return cmd
}

var AllCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Advance all eligible charges",
		RunE: func(cmd *cobra.Command, args []string) error {
			if internal.App.ChargesAutoAdvancer == nil {
				return fmt.Errorf("charges are not enabled")
			}

			var ns []string
			if namespace != "" {
				ns = append(ns, namespace)
			}

			return internal.App.ChargesAutoAdvancer.All(cmd.Context(), ns)
		},
	}

	return cmd
}
