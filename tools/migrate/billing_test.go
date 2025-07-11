package migrate_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	v20250605102416 "github.com/openmeterio/openmeter/tools/migrate/testdata/sqlcgen/20250605102416/db"
	v20250609172811 "github.com/openmeterio/openmeter/tools/migrate/testdata/sqlcgen/20250609172811/db"
)

func TestMigrateFlatFeesToUBPFlatFees(t *testing.T) {
	runner{stops{
		{
			version:   20250605102416,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				loadFixture(t, db, "testdata/sqlcgen/20250605102416/fixture.sql")
			},
		},
		{
			version:   20250605131637,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				const (
					validFlatFeeLineID                        = "01JWB2ND43KPCHMXHKD0KETF5M"
					validFlatFeeLineIDWithInvoicingExternalID = "01JWB2ND43KPCHMXHKCYEQ13WY"
					deletedFlatFeeLineID                      = "01JWB2ND43KPCHMXHKCW3PFW6W"
				)

				q := v20250605102416.New(db)
				t.Run("valid flat fee line", func(t *testing.T) {
					// Case 1: Valid flat fee line

					lines, err := q.GetLineHierarchyByDetailedLineID(t.Context(), validFlatFeeLineID)
					require.NoError(t, err)

					require.Len(t, lines.DetailedLines, 1)
					detailedLine := lines.DetailedLines[0]
					ubpLine := lines.Line

					// the ulid generated by pgsql is a valid one
					parsedID, err := ulid.Parse(ubpLine.ID)
					require.NoError(t, err)
					require.Equal(t, parsedID.String(), ubpLine.ID)

					// Let's fetch the child lines
					require.NoError(t, err)

					// Let's assert the ubp line
					require.Equal(t, detailedLine.ID, validFlatFeeLineID)
					require.Equal(t, ubpLine.Namespace, detailedLine.Namespace)

					// Field validations (no ID missmatch)
					require.NotEqual(t, detailedLine.ID, ubpLine.ID)

					// Lines are not deleted
					require.False(t, ubpLine.DeletedAt.Valid)

					// Detailed line changes
					require.Equal(t, detailedLine.ChildUniqueReferenceID.String, "flat-price")
					require.False(t, detailedLine.DeletedAt.Valid, "detailed line should not be deleted")
					require.Equal(t, detailedLine.ManagedBy, "system")
					require.False(t, detailedLine.SubscriptionID.Valid, "detailed line should not have a subscription")
					require.False(t, detailedLine.SubscriptionItemID.Valid, "detailed line should not have a subscription item")
					require.False(t, detailedLine.SubscriptionPhaseID.Valid, "detailed line should not have a subscription phase")
					require.False(t, detailedLine.Metadata.Valid, "detailed line should not have metadata")

					// UBP line changes
					require.True(t, strings.HasSuffix(ubpLine.ChildUniqueReferenceID.String, "/first-phase/in-advance/v[0]/period[2]"))

					// metadata
					metadata := map[string]string{}
					err = json.Unmarshal(ubpLine.Metadata.RawMessage, &metadata)
					require.NoError(t, err)
					require.Equal(t, metadata["/openmeter-line-reason"], "add-line-wrapping", "metadata should be added signifying why we have added this line")

					// price
					price := map[string]string{}
					err = json.Unmarshal(ubpLine.Price, &price)
					require.NoError(t, err)
					require.Equal(t, price["type"], "flat", "price type should be flat")
					require.Equal(t, price["amount"], "6", "price amount should be 6")
					require.Equal(t, price["paymentTerm"], "in_advance", "price payment term should be in_advance")

					// other fields
					require.Equal(t, ubpLine.PriceType, "flat", "price type should be flat")
					require.Equal(t, ubpLine.Quantity.String, "1", "quantity should be 1")
					require.Equal(t, ubpLine.PreLinePeriodQuantity.String, "0", "pre line period quantity should be 0")
					require.Equal(t, ubpLine.MeteredQuantity.String, "1", "metered quantity should be 1")
					require.Equal(t, ubpLine.MeteredPreLinePeriodQuantity.String, "0", "metered pre line period quantity should be 0")
					require.False(t, ubpLine.ParentLineID.Valid, "there should be no parent line for the ubp one")
					require.NotEmpty(t, ubpLine.InvoiceID, "the invoice id should be valid")

					require.Equal(t, ubpLine.InvoiceID, detailedLine.InvoiceID, "the invoice id should be the same")
					require.Equal(t, ubpLine.Amount, detailedLine.Amount)
					require.Equal(t, ubpLine.Total, detailedLine.Total)
					require.Equal(t, "6", ubpLine.Total)
					require.Equal(t, "6", ubpLine.Amount)
				})

				t.Run("valid flat fee line with invoicing external id", func(t *testing.T) {
					lines, err := q.GetLineHierarchyByDetailedLineID(t.Context(), validFlatFeeLineIDWithInvoicingExternalID)
					require.NoError(t, err)

					require.Len(t, lines.DetailedLines, 1)
					detailedLine := lines.DetailedLines[0]
					ubpLine := lines.Line

					// detailed lines must have invoicing external id set, while the ubp line should not
					require.Equal(t, detailedLine.InvoicingAppExternalID.String, "invoicing-external-id", "detailed line should have invoicing external id set")
					require.False(t, ubpLine.InvoicingAppExternalID.Valid, "ubp line should not have invoicing external id set")
				})

				t.Run("deleted flat fee line", func(t *testing.T) {
					lines, err := q.GetLineHierarchyByDetailedLineID(t.Context(), deletedFlatFeeLineID)
					require.NoError(t, err)

					require.Len(t, lines.DetailedLines, 1)
					detailedLine := lines.DetailedLines[0]
					ubpLine := lines.Line

					require.True(t, ubpLine.DeletedAt.Valid, "ubp line should be deleted")
					require.True(t, detailedLine.DeletedAt.Valid, "detailed line should be deleted")
					require.Equal(t, detailedLine.DeletedAt.Time, ubpLine.DeletedAt.Time, "detailed line and ubp line should have the same deleted at")
				})
			},
		},
	}}.Test(t)
}

func TestMigrateSplitLinesToSplitLineGroups(t *testing.T) {
	var preMigrationLineCountsByType map[string]int64
	runner{stops{
		{
			version:   20250609172811,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				loadFixture(t, db, "testdata/sqlcgen/20250609172811/fixture.sql")

				q := v20250609172811.New(db)
				lineCounts, err := q.CountLinesByStatusType(t.Context())
				require.NoError(t, err)

				preMigrationLineCountsByType = make(map[string]int64)
				for _, lineCount := range lineCounts {
					preMigrationLineCountsByType[fmt.Sprintf("%s-%s", lineCount.Status, lineCount.Type)] = lineCount.Count
				}
			},
		},
		{
			version:   20250609204117,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				const (
					SplitLineGroupID = "01JXA7Y5CRJF0NJ5ADKNZVDTGH"
				)
				q := v20250609172811.New(db)

				t.Run("split line group exists", func(t *testing.T) {
					splitLineGroup, err := q.GetSplitLineGroup(t.Context(), SplitLineGroupID)
					require.NoError(t, err)

					require.Equal(t, SplitLineGroupID, splitLineGroup.ID)
					require.Equal(t, "ns-ubp-invoicing-progressive", splitLineGroup.Namespace)
					require.Equal(t, "flat-per-unit", splitLineGroup.FeatureKey.String)
					require.Equal(t, `{"type": "unit", "amount": "100", "maximumAmount": "2000"}`, string(splitLineGroup.Price))
					require.Equal(t, "2024-09-02T12:13:00Z", splitLineGroup.ServicePeriodStart.Format(time.RFC3339))
					require.Equal(t, "2024-09-03T12:13:00Z", splitLineGroup.ServicePeriodEnd.Format(time.RFC3339))
				})

				t.Run("line counts are unchanged except of the split line", func(t *testing.T) {
					lineCounts, err := q.CountLinesByStatusType(t.Context())
					require.NoError(t, err)

					postMigrationLineCountsByType := make(map[string]int64)
					for _, lineCount := range lineCounts {
						postMigrationLineCountsByType[fmt.Sprintf("%s-%s", lineCount.Status, lineCount.Type)] = lineCount.Count
					}

					delete(preMigrationLineCountsByType, "split-usage_based")

					require.Equal(t, preMigrationLineCountsByType, postMigrationLineCountsByType)
				})

				t.Run("split line group is associated with the correct lines", func(t *testing.T) {
					splitLineGroup, err := q.GetSplitLineGroup(t.Context(), SplitLineGroupID)
					require.NoError(t, err)

					lines, err := q.GetUsageBasedLinesBySplitLineGroup(t.Context(), sql.NullString{String: SplitLineGroupID, Valid: true})
					require.NoError(t, err)

					require.Len(t, lines, 2)
					for _, line := range lines {
						require.Equal(t, SplitLineGroupID, line.SplitLineGroupID.String)
						require.Equal(t, splitLineGroup.FeatureKey.String, line.FeatureKey.String)
						require.Equal(t, splitLineGroup.Price, line.Price)
					}
				})
			},
		},
	}}.Test(t)
}

// loadFixture loads a fixture SQL file into the database
//
// If you find a better alternative, please replace this function, as it does not understand the SQL syntax, and
// it's barely enough to load the fixture.
func loadFixture(t *testing.T, db *sql.DB, fixturePath string) {
	sqlBytes, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	_, err = db.ExecContext(context.Background(), string(sqlBytes))
	require.NoError(t, err)
}
