package invoicesync

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func newTestInvoice(opts ...func(*billing.StandardInvoice)) billing.StandardInvoice {
	inv := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			ID:        "inv-test-123",
			Namespace: "ns-test",
			Currency:  currencyx.Code("USD"),
			Customer: billing.InvoiceCustomer{
				CustomerID: "cust-123",
			},
			ExternalIDs: billing.InvoiceExternalIDs{},
			Workflow: billing.InvoiceWorkflow{
				Config: billing.WorkflowConfig{
					Invoicing: billing.InvoicingConfig{
						AutoAdvance: true,
					},
					Payment: billing.PaymentConfig{
						CollectionMethod: billing.CollectionMethodChargeAutomatically,
					},
					Tax: billing.WorkflowTaxConfig{
						Enabled:  true,
						Enforced: false,
					},
				},
			},
		},
		Lines: billing.StandardInvoiceLines{
			Option: mo.Some(billing.StandardLines([]*billing.StandardLine{
				{
					StandardLineBase: billing.StandardLineBase{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{Namespace: "ns-test"},
							ID:              "line-1",
							Name:            "API Calls",
						},
					},
					DetailedLines: billing.DetailedLines{
						{
							DetailedLineBase: billing.DetailedLineBase{
								ManagedResource: models.ManagedResource{
									NamespacedModel: models.NamespacedModel{Namespace: "ns-test"},
									ID:              "dl-1",
									Name:            "API Calls",
								},
								InvoiceID: "inv-test-123",
								Currency:  currencyx.Code("USD"),
								Quantity:  alpacadecimal.NewFromInt(1),
								Totals: totals.Totals{
									Amount: alpacadecimal.NewFromFloat(100.00),
								},
								ServicePeriod: billing.Period{
									Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
									End:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
								},
							},
						},
					},
				},
			})),
		},
	}

	for _, opt := range opts {
		opt(&inv)
	}

	return inv
}

func withExternalInvoicingID(id string) func(*billing.StandardInvoice) {
	return func(inv *billing.StandardInvoice) {
		inv.ExternalIDs.Invoicing = id
	}
}

func withDetailedLineExternalID(lineIndex, detailedIndex int, externalID string) func(*billing.StandardInvoice) {
	return func(inv *billing.StandardInvoice) {
		lines := inv.Lines.OrEmpty()
		lines[lineIndex].DetailedLines[detailedIndex].ExternalIDs.Invoicing = externalID
	}
}

func strPtr(s string) *string {
	return &s
}

func TestGenerateDraftSyncPlan_Create(t *testing.T) {
	invoice := newTestInvoice()

	sessionID, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:          invoice,
		StripeCustomerID: "cus_stripe_123",
		AppID:            "app-1",
		Currency:         "USD",
	})

	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	require.Len(t, ops, 2, "create flow should produce InvoiceCreate + LineItemAdd")

	// Op 0: InvoiceCreate
	assert.Equal(t, OpTypeInvoiceCreate, ops[0].Type)
	assert.Equal(t, 0, ops[0].Sequence)
	assert.Equal(t, OpStatusPending, ops[0].Status)
	assert.NotEmpty(t, ops[0].IdempotencyKey)

	var createPayload InvoiceCreatePayload
	require.NoError(t, json.Unmarshal(ops[0].Payload, &createPayload))
	assert.Equal(t, "app-1", createPayload.AppID)
	assert.Equal(t, "ns-test", createPayload.Namespace)
	assert.Equal(t, "cust-123", createPayload.CustomerID)
	assert.Equal(t, "inv-test-123", createPayload.InvoiceID)
	assert.Equal(t, "cus_stripe_123", createPayload.StripeCustomerID)
	assert.Equal(t, "USD", createPayload.Currency)
	assert.True(t, createPayload.AutomaticTaxEnabled)

	// Op 1: LineItemAdd
	assert.Equal(t, OpTypeLineItemAdd, ops[1].Type)
	assert.Equal(t, 1, ops[1].Sequence)
	assert.NotEmpty(t, ops[1].IdempotencyKey)

	var addPayload LineItemAddPayload
	require.NoError(t, json.Unmarshal(ops[1].Payload, &addPayload))
	require.Len(t, addPayload.Lines, 1)
	assert.Equal(t, "API Calls", addPayload.Lines[0].Description)
	assert.Equal(t, int64(10000), addPayload.Lines[0].Amount) // $100.00 in cents
	assert.Equal(t, "cus_stripe_123", addPayload.Lines[0].CustomerID)
}

func TestGenerateDraftSyncPlan_CreateNoLines(t *testing.T) {
	invoice := newTestInvoice()
	// Clear lines
	invoice.Lines = billing.StandardInvoiceLines{
		Option: mo.Some(billing.StandardLines([]*billing.StandardLine{
			{
				StandardLineBase: billing.StandardLineBase{
					ManagedResource: models.ManagedResource{
						NamespacedModel: models.NamespacedModel{Namespace: "ns-test"},
						ID:              "line-empty",
						Name:            "Empty",
					},
				},
				DetailedLines: billing.DetailedLines{},
			},
		})),
	}

	sessionID, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:          invoice,
		StripeCustomerID: "cus_stripe_123",
		AppID:            "app-1",
		Currency:         "USD",
	})

	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	// Only InvoiceCreate, no line add since no leaf lines
	require.Len(t, ops, 1)
	assert.Equal(t, OpTypeInvoiceCreate, ops[0].Type)
}

func TestGenerateDraftSyncPlan_Update(t *testing.T) {
	// Invoice already has a Stripe ID and line with external ID
	invoice := newTestInvoice(
		withExternalInvoicingID("in_stripe_123"),
		withDetailedLineExternalID(0, 0, "il_stripe_456"),
	)

	existingStripeLines := []*stripe.InvoiceLineItem{
		{
			ID: "il_stripe_456",
			InvoiceItem: &stripe.InvoiceItem{
				ID: "ii_stripe_456",
			},
		},
	}

	sessionID, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:             invoice,
		StripeCustomerID:    "cus_stripe_123",
		AppID:               "app-1",
		Currency:            "USD",
		ExistingStripeLines: existingStripeLines,
	})

	require.NoError(t, err)
	require.NotEmpty(t, sessionID)

	// Should have: InvoiceUpdate + LineItemUpdate (existing line)
	var hasInvoiceUpdate, hasLineItemUpdate bool
	for _, op := range ops {
		switch op.Type {
		case OpTypeInvoiceUpdate:
			hasInvoiceUpdate = true
		case OpTypeLineItemUpdate:
			hasLineItemUpdate = true
		}
	}
	require.True(t, hasInvoiceUpdate, "ops should include InvoiceUpdate")
	require.True(t, hasLineItemUpdate, "ops should include LineItemUpdate")

	// First op should be InvoiceUpdate
	assert.Equal(t, OpTypeInvoiceUpdate, ops[0].Type)
	assert.Equal(t, 0, ops[0].Sequence)

	var updatePayload InvoiceUpdatePayload
	require.NoError(t, json.Unmarshal(ops[0].Payload, &updatePayload))
	assert.Equal(t, "in_stripe_123", updatePayload.StripeInvoiceID)
	assert.True(t, updatePayload.AutomaticTaxEnabled)

	// Should have a LineItemUpdate for the existing line
	foundUpdate := false
	for _, op := range ops {
		if op.Type == OpTypeLineItemUpdate {
			foundUpdate = true
			var linePayload LineItemUpdatePayload
			require.NoError(t, json.Unmarshal(op.Payload, &linePayload))
			assert.Equal(t, "in_stripe_123", linePayload.StripeInvoiceID)
			require.Len(t, linePayload.Lines, 1)
			assert.Equal(t, "il_stripe_456", linePayload.Lines[0].ID)
		}
	}
	assert.True(t, foundUpdate, "should have a LineItemUpdate operation")
}

// When OpenMeter still references a Stripe line ID that is not present on the draft invoice,
// the planner must not skip the line; it should add it (same as no external ID) and must not
// clear unrelated lines from the removal set for the missing ID.
func TestGenerateDraftSyncPlan_UpdateWhenStripeLineMissingForExternalID(t *testing.T) {
	invoice := newTestInvoice(
		withExternalInvoicingID("in_stripe_123"),
		withDetailedLineExternalID(0, 0, "il_missing_on_stripe"),
	)

	existingStripeLines := []*stripe.InvoiceLineItem{
		{
			ID:          "il_unrelated",
			InvoiceItem: &stripe.InvoiceItem{ID: "ii_unrelated"},
		},
	}

	_, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:             invoice,
		StripeCustomerID:    "cus_stripe_123",
		AppID:               "app-1",
		Currency:            "USD",
		ExistingStripeLines: existingStripeLines,
	})
	require.NoError(t, err)

	var removePayload LineItemRemovePayload
	var addPayload LineItemAddPayload
	for _, op := range ops {
		switch op.Type {
		case OpTypeLineItemRemove:
			require.NoError(t, json.Unmarshal(op.Payload, &removePayload))
		case OpTypeLineItemAdd:
			require.NoError(t, json.Unmarshal(op.Payload, &addPayload))
		}
	}

	require.Len(t, removePayload.LineIDs, 1, "unrelated Stripe line should still be removed")
	assert.Equal(t, "il_unrelated", removePayload.LineIDs[0])
	require.NotEmpty(t, addPayload.Lines, "line with stale external ID should be added, not skipped")
}

func TestGenerateDraftSyncPlan_UpdateWhenDiscountStripeLineMissingForExternalID(t *testing.T) {
	taxBehavior := productcatalog.InclusiveTaxBehavior

	invoice := newTestInvoice(
		withExternalInvoicingID("in_stripe_123"),
		withDetailedLineExternalID(0, 0, "il_line_1"),
	)

	lines := invoice.Lines.OrEmpty()
	lines[0].DetailedLines[0].TaxConfig = &productcatalog.TaxConfig{
		Behavior: &taxBehavior,
		Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
	}
	lines[0].DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{
		{
			ManagedModelWithID: models.ManagedModelWithID{ID: "disc-1"},
			AmountLineDiscount: billing.AmountLineDiscount{
				LineDiscountBase: billing.LineDiscountBase{
					Description: strPtr("20% off"),
					ExternalIDs: billing.LineExternalIDs{Invoicing: "il_disc_missing"},
				},
				Amount: alpacadecimal.NewFromFloat(20.00),
			},
		},
	}

	// Only the main line exists on Stripe; discount ID is stale.
	existingStripeLines := []*stripe.InvoiceLineItem{
		{ID: "il_line_1", InvoiceItem: &stripe.InvoiceItem{ID: "ii_line_1"}},
	}

	_, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:             invoice,
		StripeCustomerID:    "cus_stripe_123",
		AppID:               "app-1",
		Currency:            "USD",
		ExistingStripeLines: existingStripeLines,
	})
	require.NoError(t, err)

	var updatePayload LineItemUpdatePayload
	var addPayload LineItemAddPayload
	for _, op := range ops {
		switch op.Type {
		case OpTypeLineItemUpdate:
			require.NoError(t, json.Unmarshal(op.Payload, &updatePayload))
		case OpTypeLineItemAdd:
			require.NoError(t, json.Unmarshal(op.Payload, &addPayload))
		}
	}

	require.Len(t, updatePayload.Lines, 1, "only the regular line should be updated")
	assert.Equal(t, "il_line_1", updatePayload.Lines[0].ID)

	require.Len(t, addPayload.Lines, 1, "stale discount external ID should produce an add, not be skipped")
	assert.Equal(t, LineMetadataTypeDiscount, addPayload.Lines[0].Metadata[LineMetadataType])
}

func TestGenerateDraftSyncPlan_UpdateWithNewAndRemovedLines(t *testing.T) {
	// Existing invoice with one line, Stripe has an extra line to remove
	invoice := newTestInvoice(
		withExternalInvoicingID("in_stripe_123"),
	)
	// The detailed line has no external ID — it's new

	existingStripeLines := []*stripe.InvoiceLineItem{
		{
			ID: "il_old_line",
			InvoiceItem: &stripe.InvoiceItem{
				ID: "ii_old_line",
			},
		},
	}

	_, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:             invoice,
		StripeCustomerID:    "cus_stripe_123",
		AppID:               "app-1",
		Currency:            "USD",
		ExistingStripeLines: existingStripeLines,
	})

	require.NoError(t, err)

	// Should have: InvoiceUpdate, LineItemRemove (old line), LineItemAdd (new line)
	opTypes := map[OpType]bool{}
	for _, op := range ops {
		opTypes[op.Type] = true
	}

	assert.True(t, opTypes[OpTypeInvoiceUpdate], "should have InvoiceUpdate")
	assert.True(t, opTypes[OpTypeLineItemRemove], "should have LineItemRemove for old line")
	assert.True(t, opTypes[OpTypeLineItemAdd], "should have LineItemAdd for new line")

	// Verify ordering: remove should come before add
	var removeSeq, addSeq int
	for _, op := range ops {
		if op.Type == OpTypeLineItemRemove {
			removeSeq = op.Sequence
		}
		if op.Type == OpTypeLineItemAdd {
			addSeq = op.Sequence
		}
	}
	assert.Less(t, removeSeq, addSeq, "remove should happen before add")
}

func TestGenerateDraftSyncPlan_IdempotencyKeys(t *testing.T) {
	invoice := newTestInvoice()

	sessionID1, ops1, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:          invoice,
		StripeCustomerID: "cus_stripe_123",
		AppID:            "app-1",
		Currency:         "USD",
	})
	require.NoError(t, err)

	// Same input, same session should produce same keys
	ops2 := make([]SyncOperation, len(ops1))
	for i, op := range ops1 {
		ops2[i] = SyncOperation{
			Sequence:       op.Sequence,
			Type:           op.Type,
			IdempotencyKey: GenerateIdempotencyKey(invoice.ID, sessionID1, op.Sequence, op.Type),
		}
	}

	for i := range ops1 {
		assert.Equal(t, ops1[i].IdempotencyKey, ops2[i].IdempotencyKey,
			"same session should produce same keys for op %d", i)
	}

	// Different session should produce different keys
	_, ops3, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:          invoice,
		StripeCustomerID: "cus_stripe_123",
		AppID:            "app-1",
		Currency:         "USD",
	})
	require.NoError(t, err)

	// Session IDs differ because GenerateDraftSyncPlan generates a new one each time
	for i := range ops1 {
		assert.NotEqual(t, ops1[i].IdempotencyKey, ops3[i].IdempotencyKey,
			"different sessions should produce different keys for op %d", i)
	}
}

func TestGenerateIssuingSyncPlan(t *testing.T) {
	invoice := newTestInvoice(
		withExternalInvoicingID("in_stripe_123"),
		withDetailedLineExternalID(0, 0, "il_stripe_456"),
	)

	existingStripeLines := []*stripe.InvoiceLineItem{
		{
			ID: "il_stripe_456",
			InvoiceItem: &stripe.InvoiceItem{
				ID: "ii_stripe_456",
			},
		},
	}

	sessionID, ops, err := GenerateIssuingSyncPlan(PlanGeneratorInput{
		Invoice:             invoice,
		StripeCustomerID:    "cus_stripe_123",
		AppID:               "app-1",
		Currency:            "USD",
		ExistingStripeLines: existingStripeLines,
	})

	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	require.GreaterOrEqual(t, len(ops), 2, "issuing should have at least update + finalize")

	// Last op should be InvoiceFinalize
	lastOp := ops[len(ops)-1]
	assert.Equal(t, OpTypeInvoiceFinalize, lastOp.Type)

	var finalizePayload InvoiceFinalizePayload
	require.NoError(t, json.Unmarshal(lastOp.Payload, &finalizePayload))
	assert.Equal(t, "in_stripe_123", finalizePayload.StripeInvoiceID)
	assert.True(t, finalizePayload.AutoAdvance)

	// Verify sequences are monotonically increasing
	for i := 1; i < len(ops); i++ {
		assert.Greater(t, ops[i].Sequence, ops[i-1].Sequence,
			"sequences should be monotonically increasing")
	}
}

func TestGenerateIssuingSyncPlan_NoExternalID(t *testing.T) {
	invoice := newTestInvoice() // no external ID

	_, _, err := GenerateIssuingSyncPlan(PlanGeneratorInput{
		Invoice:          invoice,
		StripeCustomerID: "cus_stripe_123",
		AppID:            "app-1",
		Currency:         "USD",
	})

	require.Error(t, err, "issuing without external ID should fail")
	assert.Contains(t, err.Error(), "no Stripe external ID")
}

func TestGenerateDeleteSyncPlan(t *testing.T) {
	t.Run("with external ID", func(t *testing.T) {
		invoice := newTestInvoice(withExternalInvoicingID("in_stripe_123"))

		sessionID, ops, err := GenerateDeleteSyncPlan(PlanGeneratorInput{
			Invoice: invoice,
			AppID:   "app-1",
		})

		require.NoError(t, err)
		require.NotEmpty(t, sessionID)
		require.Len(t, ops, 1)

		assert.Equal(t, OpTypeInvoiceDelete, ops[0].Type)
		assert.Equal(t, 0, ops[0].Sequence)

		var deletePayload InvoiceDeletePayload
		require.NoError(t, json.Unmarshal(ops[0].Payload, &deletePayload))
		assert.Equal(t, "in_stripe_123", deletePayload.StripeInvoiceID)
	})

	t.Run("without external ID", func(t *testing.T) {
		invoice := newTestInvoice()

		sessionID, ops, err := GenerateDeleteSyncPlan(PlanGeneratorInput{
			Invoice: invoice,
			AppID:   "app-1",
		})

		require.NoError(t, err)
		require.NotEmpty(t, sessionID)
		require.Len(t, ops, 0, "no ops needed when no Stripe invoice exists")
	})
}

func TestGenerateDraftSyncPlan_WithDiscountsAndTax(t *testing.T) {
	taxBehavior := productcatalog.ExclusiveTaxBehavior

	invoice := newTestInvoice()
	lines := invoice.Lines.OrEmpty()
	lines[0].DetailedLines[0].TaxConfig = &productcatalog.TaxConfig{
		Behavior: &taxBehavior,
		Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
	}
	lines[0].DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{
		{
			ManagedModelWithID: models.ManagedModelWithID{
				ID: "disc-1",
			},
			AmountLineDiscount: billing.AmountLineDiscount{
				LineDiscountBase: billing.LineDiscountBase{
					Description: strPtr("10% off"),
				},
				Amount: alpacadecimal.NewFromFloat(10.00),
			},
		},
	}

	_, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:          invoice,
		StripeCustomerID: "cus_stripe_123",
		AppID:            "app-1",
		Currency:         "USD",
	})

	require.NoError(t, err)
	require.Len(t, ops, 2) // InvoiceCreate + LineItemAdd

	// LineItemAdd should include both the line and the discount
	var addPayload LineItemAddPayload
	require.NoError(t, json.Unmarshal(ops[1].Payload, &addPayload))
	require.Len(t, addPayload.Lines, 2, "should have regular line + discount line")

	// Find the discount line
	var discountLine *LineItemParams
	var regularLine *LineItemParams
	for i := range addPayload.Lines {
		if addPayload.Lines[i].Metadata[LineMetadataType] == LineMetadataTypeDiscount {
			discountLine = &addPayload.Lines[i]
		} else {
			regularLine = &addPayload.Lines[i]
		}
	}

	require.NotNil(t, discountLine, "should have a discount line")
	require.NotNil(t, regularLine, "should have a regular line")

	assert.Less(t, discountLine.Amount, int64(0), "discount should be negative")
	assert.Equal(t, "disc-1", discountLine.Metadata[LineMetadataID])

	// Tax should be set on both lines
	assert.NotNil(t, discountLine.TaxBehavior)
	assert.Equal(t, "exclusive", *discountLine.TaxBehavior)
	assert.NotNil(t, discountLine.TaxCode)
	assert.Equal(t, "txcd_10000000", *discountLine.TaxCode)

	assert.NotNil(t, regularLine.TaxBehavior)
	assert.Equal(t, "exclusive", *regularLine.TaxBehavior)
	assert.NotNil(t, regularLine.TaxCode)
	assert.Equal(t, "txcd_10000000", *regularLine.TaxCode)
}

func TestGenerateDraftSyncPlan_UpdateWithExistingDiscounts(t *testing.T) {
	// Invoice with a line and discount that both have external IDs (update path)
	taxBehavior := productcatalog.InclusiveTaxBehavior

	invoice := newTestInvoice(
		withExternalInvoicingID("in_stripe_123"),
		withDetailedLineExternalID(0, 0, "il_line_1"),
	)

	// Add discount with external ID to the detailed line
	lines := invoice.Lines.OrEmpty()
	lines[0].DetailedLines[0].TaxConfig = &productcatalog.TaxConfig{
		Behavior: &taxBehavior,
		Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
	}
	lines[0].DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{
		{
			ManagedModelWithID: models.ManagedModelWithID{ID: "disc-1"},
			AmountLineDiscount: billing.AmountLineDiscount{
				LineDiscountBase: billing.LineDiscountBase{
					Description: strPtr("20% off"),
					ExternalIDs: billing.LineExternalIDs{Invoicing: "il_disc_1"},
				},
				Amount: alpacadecimal.NewFromFloat(20.00),
			},
		},
	}

	existingStripeLines := []*stripe.InvoiceLineItem{
		{ID: "il_line_1", InvoiceItem: &stripe.InvoiceItem{ID: "ii_line_1"}},
		{ID: "il_disc_1", InvoiceItem: &stripe.InvoiceItem{ID: "ii_disc_1"}},
	}

	_, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:             invoice,
		StripeCustomerID:    "cus_stripe_123",
		AppID:               "app-1",
		Currency:            "USD",
		ExistingStripeLines: existingStripeLines,
	})
	require.NoError(t, err)

	// Should have: InvoiceUpdate + LineItemUpdate (for existing line + discount)
	foundUpdate := false
	for _, op := range ops {
		if op.Type == OpTypeLineItemUpdate {
			foundUpdate = true
			var payload LineItemUpdatePayload
			require.NoError(t, json.Unmarshal(op.Payload, &payload))

			// Should update both the line and the discount
			assert.Len(t, payload.Lines, 2, "should update line + discount")

			// Find the discount update
			for _, line := range payload.Lines {
				if line.Metadata[LineMetadataType] == LineMetadataTypeDiscount {
					assert.Equal(t, "il_disc_1", line.ID)
					assert.Less(t, line.Amount, int64(0), "discount should be negative")
					// Tax should be applied via setTax on update params
					assert.NotNil(t, line.TaxBehavior)
					assert.Equal(t, "inclusive", *line.TaxBehavior)
					assert.NotNil(t, line.TaxCode)
					assert.Equal(t, "txcd_20000000", *line.TaxCode)
				}
			}
		}
	}
	assert.True(t, foundUpdate, "should have LineItemUpdate for existing lines")
}

func TestAllOperationsHaveUniqueKeys(t *testing.T) {
	invoice := newTestInvoice(
		withExternalInvoicingID("in_stripe_123"),
	)

	existingStripeLines := []*stripe.InvoiceLineItem{
		{
			ID: "il_old",
			InvoiceItem: &stripe.InvoiceItem{
				ID: "ii_old",
			},
		},
	}

	_, ops, err := GenerateDraftSyncPlan(PlanGeneratorInput{
		Invoice:             invoice,
		StripeCustomerID:    "cus_stripe_123",
		AppID:               "app-1",
		Currency:            "USD",
		ExistingStripeLines: existingStripeLines,
	})
	require.NoError(t, err)

	keys := map[string]bool{}
	for _, op := range ops {
		assert.False(t, keys[op.IdempotencyKey], "duplicate idempotency key found: %s", op.IdempotencyKey)
		keys[op.IdempotencyKey] = true
	}
}
