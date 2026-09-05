package ledger_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestOriginIdentityVersionAndPersistenceContract(t *testing.T) {
	origin := "01J00000000000000000000001"
	spend := "01J00000000000000000000002"
	source := "01J00000000000000000000003"
	parts := ledger.EntryIdentityParts{OriginID: &origin, SpendChargeID: &spend, SourceChargeID: &source, CorrectionSource: lo.ToPtr("entry:original")}
	key, version := parts.Text()
	require.Equal(t, ledger.EntryIdentityVersion3, version)
	parsedVersion, parsed, err := key.Parse()
	require.NoError(t, err)
	require.Equal(t, version, parsedVersion)
	require.Equal(t, parts, parsed)
	entry := validationEntryInput{identityKey: string(key), schemaVersion: ledger.EntrySchemaVersionOrigin, originID: &origin, spendChargeID: &spend, sourceChargeID: &source}
	require.NoError(t, ledger.ValidateEntryIdentityKey(entry))

	for _, test := range []struct {
		name    string
		mutate  func(*validationEntryInput)
		message string
	}{
		{"origin column dropped", func(e *validationEntryInput) { e.originID = nil }, "origin"},
		{"origin column mismatched", func(e *validationEntryInput) { e.originID = lo.ToPtr("01J00000000000000000000004") }, "does not match"},
		{"new identity in old schema", func(e *validationEntryInput) { e.schemaVersion = ledger.EntrySchemaVersionCurrent }, "origin"},
		{"empty origin", func(e *validationEntryInput) { e.originID = lo.ToPtr("") }, "origin"},
		{"invalid origin", func(e *validationEntryInput) { e.originID = lo.ToPtr("not-an-origin") }, "origin"},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := entry
			test.mutate(&copy)
			require.ErrorContains(t, ledger.ValidateEntryIdentityKey(copy), test.message)
		})
	}
	_, _, err = ledger.EntryIdentityKeyText("entry-identity:v3:too|few").Parse()
	require.Error(t, err)
}

func TestOriginProvenanceCannotLeakBetweenBalancedPairs(t *testing.T) {
	origin := "01J00000000000000000000001"
	spend := "01J00000000000000000000002"
	source := "01J00000000000000000000003"
	route := ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD"))}
	debit := validationEntryInput{
		amount: alpacadecimal.NewFromInt(-10), originID: &origin, spendChargeID: &spend, sourceChargeID: &source,
		address: testEntryIdentityAddress(t, ledger.AccountTypeCustomerAccrued, "accrued", route),
	}
	credit := debit
	credit.amount = debit.amount.Neg()
	credit.address = testEntryIdentityAddress(t, ledger.AccountTypeEarnings, "earnings", route)
	require.NoError(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{debit, credit}))
	for _, test := range []struct {
		name    string
		mutate  func(*validationEntryInput)
		message string
	}{
		{"dropped origin", func(e *validationEntryInput) { e.originID = nil }, "balance independently"},
		{"different origin", func(e *validationEntryInput) { e.originID = lo.ToPtr("01J00000000000000000000004") }, "balance independently"},
		{"dropped spend", func(e *validationEntryInput) { e.spendChargeID = nil }, "spend_charge_id"},
		{"different spend", func(e *validationEntryInput) { e.spendChargeID = lo.ToPtr("another-spend") }, "preserve spend"},
		{"different purchase", func(e *validationEntryInput) { e.sourceChargeID = lo.ToPtr("another-source") }, "preserve source"},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := credit
			test.mutate(&copy)
			require.ErrorContains(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{debit, copy}), test.message)
		})
	}
	// Attribution can change source within accrued without moving value to another origin.
	translated := credit
	translated.address = debit.address
	translated.sourceChargeID = lo.ToPtr("another-source")
	require.ErrorContains(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{debit, translated}), "attribute unknown")
	unknownDebit := debit
	unknownDebit.sourceChargeID = nil
	require.NoError(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{unknownDebit, translated}))
	// One recognition can contain distinct backing sources of the same advance.
	secondDebit, secondCredit := debit, credit
	secondDebit.sourceChargeID, secondCredit.sourceChargeID = translated.sourceChargeID, translated.sourceChargeID
	require.NoError(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{debit, credit, secondDebit, secondCredit}))
}
