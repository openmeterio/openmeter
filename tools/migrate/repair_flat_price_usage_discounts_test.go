package migrate_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestRepairFlatPriceUsageDiscountsMigration(t *testing.T) {
	const (
		flatPriceJSON      = `{"type":"flat","amount":"100","paymentTerm":"in_advance"}`
		unitPriceJSON      = `{"type":"unit","amount":"1"}`
		mixedDiscountsJSON = `{
			"percentage":{"percentage":50,"correlationID":"percentage-correlation"},
			"usage":{"quantity":"10","correlationID":"usage-correlation"}
		}`
		percentageDiscountJSON = `{
			"percentage":{"percentage":50,"correlationID":"percentage-correlation"}
		}`
		usageDiscountJSON = `{
			"usage":{"quantity":"10","correlationID":"usage-correlation"}
		}`
	)

	now := time.Date(2026, 7, 16, 12, 14, 5, 0, time.UTC)
	deletedAt := now.Add(time.Hour)
	namespace := "default"
	gatheringInvoiceID := ulid.Make().String()

	testCases := []repairFlatPriceUsageDiscountCase{
		{
			name:          "flat price preserves percentage and removes usage",
			priceType:     "flat",
			price:         flatPriceJSON,
			discounts:     mixedDiscountsJSON,
			wantDiscounts: percentageDiscountJSON,
		},
		{
			name:      "flat price with usage only becomes null",
			priceType: "flat",
			price:     flatPriceJSON,
			discounts: usageDiscountJSON,
			wantNull:  true,
		},
		{
			name:          "unit price usage discount is unchanged",
			priceType:     "unit",
			price:         unitPriceJSON,
			discounts:     mixedDiscountsJSON,
			wantDiscounts: mixedDiscountsJSON,
		},
		{
			name:          "flat price percentage discount is unchanged",
			priceType:     "flat",
			price:         flatPriceJSON,
			discounts:     percentageDiscountJSON,
			wantDiscounts: percentageDiscountJSON,
		},
		{
			name:          "deleted flat price line is repaired",
			priceType:     "flat",
			price:         flatPriceJSON,
			discounts:     mixedDiscountsJSON,
			deletedAt:     &deletedAt,
			wantDiscounts: percentageDiscountJSON,
		},
	}

	for idx := range testCases {
		testCases[idx].lineID = ulid.Make().String()
		testCases[idx].configID = ulid.Make().String()
	}

	base := repairUsageBasedDeletedInvoiceRunsBase{
		namespace:                namespace,
		now:                      now,
		customerID:               ulid.Make().String(),
		taxCodeID:                ulid.Make().String(),
		featureID:                ulid.Make().String(),
		workflowID:               ulid.Make().String(),
		profileID:                ulid.Make().String(),
		taxAppID:                 ulid.Make().String(),
		invoicingAppID:           ulid.Make().String(),
		paymentAppID:             ulid.Make().String(),
		gatheringInvoiceID:       gatheringInvoiceID,
		gatheringWorkflowID:      ulid.Make().String(),
		emptyGatheringCustomerID: ulid.Make().String(),
		emptyGatheringInvoiceID:  ulid.Make().String(),
		emptyGatheringWorkflowID: ulid.Make().String(),
	}

	runner{
		stops: stops{
			{
				version:   20260714144104,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					seedRepairUsageBasedDeletedInvoiceRunsBase(t, db, base)
					for _, testCase := range testCases {
						seedRepairFlatPriceUsageDiscountCase(t, db, namespace, gatheringInvoiceID, now, testCase)
					}
				},
			},
			{
				version:   20260716121405,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					for _, testCase := range testCases {
						t.Run(testCase.name, func(t *testing.T) {
							assertRepairFlatPriceUsageDiscountCase(t, db, testCase)
						})
					}

					result, err := db.Exec(`
						UPDATE billing_invoice_lines l
						SET ratecard_discounts = NULLIF(l.ratecard_discounts - 'usage', '{}'::jsonb)
						FROM billing_invoice_usage_based_line_configs u
						WHERE u.namespace = l.namespace
						  AND u.id = l.usage_based_line_config_id
						  AND u.price_type = 'flat'
						  AND l.ratecard_discounts ? 'usage'
					`)
					require.NoError(t, err)
					rowsAffected, err := result.RowsAffected()
					require.NoError(t, err)
					require.Zero(t, rowsAffected)
				},
			},
		},
	}.Test(t)
}

type repairFlatPriceUsageDiscountCase struct {
	name          string
	lineID        string
	configID      string
	priceType     string
	price         string
	discounts     string
	deletedAt     *time.Time
	wantDiscounts string
	wantNull      bool
}

func seedRepairFlatPriceUsageDiscountCase(
	t *testing.T,
	db *sql.DB,
	namespace string,
	invoiceID string,
	now time.Time,
	testCase repairFlatPriceUsageDiscountCase,
) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO billing_invoice_usage_based_line_configs (
			id, namespace, price_type, price
		) VALUES (
			$1, $2, $3, $4::jsonb
		)
	`, testCase.configID, namespace, testCase.priceType, testCase.price)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO billing_invoice_lines (
			id, namespace, metadata, created_at, updated_at, deleted_at, name,
			period_start, period_end, invoice_at, type, status, currency, invoice_id,
			managed_by, amount, taxes_total, taxes_inclusive_total, taxes_exclusive_total,
			charges_total, discounts_total, credits_total, total, engine,
			ratecard_discounts, usage_based_line_config_id
		) VALUES (
			$1, $2, '{}'::jsonb, $3, $3, $4, $5,
			$3, $6, $6, 'usage_based', 'valid', 'EUR', $7,
			'manual', 0, 0, 0, 0,
			0, 0, 0, 0, 'invoicing',
			$8::jsonb, $9
		)
	`, testCase.lineID, namespace, now, testCase.deletedAt, testCase.name, now.Add(time.Hour), invoiceID, testCase.discounts, testCase.configID)
	require.NoError(t, err)
}

func assertRepairFlatPriceUsageDiscountCase(t *testing.T, db *sql.DB, testCase repairFlatPriceUsageDiscountCase) {
	t.Helper()

	var discounts sql.NullString
	err := db.QueryRow(`
		SELECT ratecard_discounts::text
		FROM billing_invoice_lines
		WHERE id = $1
	`, testCase.lineID).Scan(&discounts)
	require.NoError(t, err)

	if testCase.wantNull {
		require.False(t, discounts.Valid)
		return
	}

	require.True(t, discounts.Valid)
	require.JSONEq(t, testCase.wantDiscounts, discounts.String)
}
