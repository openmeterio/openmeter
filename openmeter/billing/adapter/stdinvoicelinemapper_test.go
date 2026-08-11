package billingadapter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
)

func TestMapStandardInvoiceDetailedLineFromDBWithoutFlatFeeConfig(t *testing.T) {
	// given:
	// - a legacy detailed invoice line whose flat-fee config edge is not loaded
	// when:
	// - the detailed line is mapped
	// then:
	// - mapping succeeds and the flat-fee config ID remains empty
	parentLineID := "parent-line-id"
	dbLine := &db.BillingInvoiceLine{
		ID:           "detailed-line-id",
		Namespace:    "namespace",
		InvoiceID:    "invoice-id",
		ParentLineID: &parentLineID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		PeriodStart:  time.Now(),
		PeriodEnd:    time.Now(),
	}

	line, err := (&adapter{}).mapStandardInvoiceDetailedLineFromDB(dbLine)

	require.NoError(t, err)
	require.Empty(t, line.FeeLineConfigID)
}
