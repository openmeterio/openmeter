package creditsbillingcompat

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type TaxCodeDualWriteTestSuite struct {
	BaseSuite
}

func TestTaxCodeDualWrite(t *testing.T) {
	suite.Run(t, new(TaxCodeDualWriteTestSuite))
}

// assertInvoiceLineTaxCode verifies that an invoice line has a fully-stamped TaxCode: both
// the FK (TaxCodeID) and the entity snapshot (TaxCode) are present with the expected code.
func (s *TaxCodeDualWriteTestSuite) assertInvoiceLineTaxCode(line *billing.StandardLine, wantCode string) {
	s.T().Helper()
	s.Require().NotNil(line.TaxConfig)
	s.Require().NotNil(line.TaxConfig.TaxCodeID, "TaxCodeID must be stamped on line")
	s.Require().NotNil(line.TaxConfig.TaxCode, "TaxCode entity must be stamped on line")
	s.Require().NotNil(line.TaxConfig.Stripe)
	s.Equal(wantCode, line.TaxConfig.Stripe.Code)
	mapping, ok := line.TaxConfig.TaxCode.GetAppMapping(app.AppTypeStripe)
	s.True(ok, "TaxCode must have a Stripe app mapping")
	s.Equal(wantCode, mapping.TaxCode)
}

// D1: Profile DefaultTaxConfig is merged into a nil-TaxConfig line and entity is stamped.
func (s *TaxCodeDualWriteTestSuite) TestSnapshotTaxCodeIntoLinesOnAdvance() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
	})
	cust := s.CreateTestCustomer(ns, "test")
	s.SetupAccounts(ns, cust.ID)
	s.ProvisionDefaultTaxCodes(ctx, ns)

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.Charges.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Namespace:     ns,
				Period:        timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
				InvoiceAt:     now,
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "nil-taxconfig line",
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		}),
	})
	s.NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)

	s.assertInvoiceLineTaxCode(lines[0], "txcd_10000000")
	s.Require().NotNil(lines[0].TaxConfig.Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *lines[0].TaxConfig.Behavior)
}

// D2: A line's own Stripe code takes precedence over the profile DefaultTaxConfig.
func (s *TaxCodeDualWriteTestSuite) TestSnapshotLineOwnCodeTakesPrecedence() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxConfig{
		Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
	})
	cust := s.CreateTestCustomer(ns, "test")
	s.SetupAccounts(ns, cust.ID)
	s.ProvisionDefaultTaxCodes(ctx, ns)

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.Charges.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: ns,
						Name:      "line with own tax code",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
					InvoiceAt:     now,
					ManagedBy:     billing.ManuallyManagedLine,
					TaxConfig: &productcatalog.TaxConfig{
						Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
					},
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					})),
				},
			},
		}),
	})
	s.NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)

	s.assertInvoiceLineTaxCode(lines[0], "txcd_20000000")
}

// D3: A line that already has TaxCodeID pre-stamped keeps it (subscription sync scenario).
func (s *TaxCodeDualWriteTestSuite) TestSnapshotPreservesExistingTaxCodeID() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("ns-taxcode-dw")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Seed a profile with the Stripe code via the adapter to materialize the TaxCode entity.
	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	seeded := s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxConfig{
		Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
	})

	existingID := seeded.WorkflowConfig.Invoicing.DefaultTaxConfig.TaxCodeID
	s.Require().NotNil(existingID)

	cust := s.CreateTestCustomer(ns, "test")
	s.SetupAccounts(ns, cust.ID)
	s.ProvisionDefaultTaxCodes(ctx, ns)
	now := time.Now().Truncate(time.Microsecond).UTC()

	// Create a pending line with the TaxCodeID already stamped (as subscription sync would do).
	_, err := s.Charges.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: ns,
						Name:      "pre-stamped line",
					}),
					ServicePeriod: timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour * 24)},
					InvoiceAt:     now,
					ManagedBy:     billing.ManuallyManagedLine,
					TaxConfig: &productcatalog.TaxConfig{
						Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
						TaxCodeID: existingID,
					},
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					})),
				},
			},
		}),
	})
	s.NoError(err)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(invoices, 1)

	lines := invoices[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)

	s.Require().NotNil(lines[0].TaxConfig)
	s.Require().NotNil(lines[0].TaxConfig.TaxCodeID)
	s.Equal(*existingID, *lines[0].TaxConfig.TaxCodeID, "pre-stamped TaxCodeID must not be overwritten")
	s.NotNil(lines[0].TaxConfig.TaxCode, "entity snapshot should still be stamped from the deps map")
}
