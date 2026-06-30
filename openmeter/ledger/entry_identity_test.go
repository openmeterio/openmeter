package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEntryIdentityParts_Text(t *testing.T) {
	sourceChargeID := "01JABCDEF0123456789ABCDEFG"
	spendChargeID := "01JBCDEFG0123456789ABCDEFG"
	collectionSource := "0"
	correctionSource := "entry:1"

	tests := []struct {
		name            string
		parts           ledger.EntryIdentityParts
		expectedText    ledger.EntryIdentityKeyText
		expectedVersion ledger.EntryIdentityVersion
	}{
		{
			name:            "empty v1",
			expectedText:    "",
			expectedVersion: ledger.EntryIdentityVersion1,
		},
		{
			name: "collection source v1",
			parts: ledger.EntryIdentityParts{
				CollectionSource: &collectionSource,
			},
			expectedText:    "collection-source:0",
			expectedVersion: ledger.EntryIdentityVersion1,
		},
		{
			name: "correction source v1",
			parts: ledger.EntryIdentityParts{
				CorrectionSource: &correctionSource,
			},
			expectedText:    "correction-source:entry:1",
			expectedVersion: ledger.EntryIdentityVersion1,
		},
		{
			name: "source charge v2",
			parts: ledger.EntryIdentityParts{
				SourceChargeID: &sourceChargeID,
			},
			expectedText:    "entry-identity:v2:||01JABCDEF0123456789ABCDEFG|",
			expectedVersion: ledger.EntryIdentityVersion2,
		},
		{
			name: "collection source plus charge provenance v2",
			parts: ledger.EntryIdentityParts{
				CollectionSource: &collectionSource,
				SourceChargeID:   &sourceChargeID,
				SpendChargeID:    &spendChargeID,
			},
			expectedText:    "entry-identity:v2:0||01JABCDEF0123456789ABCDEFG|01JBCDEFG0123456789ABCDEFG",
			expectedVersion: ledger.EntryIdentityVersion2,
		},
		{
			name: "correction source plus spend provenance v2",
			parts: ledger.EntryIdentityParts{
				CorrectionSource: &correctionSource,
				SpendChargeID:    &spendChargeID,
			},
			expectedText:    "entry-identity:v2:|entry%3A1||01JBCDEFG0123456789ABCDEFG",
			expectedVersion: ledger.EntryIdentityVersion2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualText, actualVersion := tt.parts.Text()
			require.Equal(t, tt.expectedText, actualText)
			require.Equal(t, tt.expectedVersion, actualVersion)
			require.Equal(t, tt.expectedVersion, actualText.Version())

			parsedVersion, parsedParts, err := actualText.Parse()
			require.NoError(t, err)
			require.Equal(t, tt.expectedVersion, parsedVersion)
			require.Equal(t, tt.parts, parsedParts)
		})
	}
}

func TestEntryIdentityKeyText_Parse(t *testing.T) {
	t.Run("returns malformed v2 error", func(t *testing.T) {
		_, _, err := ledger.EntryIdentityKeyText("entry-identity:v2:too|few").Parse()
		require.ErrorContains(t, err, "invalid ledger entry identity key format")
	})
}

func TestValidateEntryIdentityKey(t *testing.T) {
	sourceChargeID := "01JABCDEF0123456789ABCDEFG"
	spendChargeID := "01JBCDEFG0123456789ABCDEFG"
	collectionSource := "0"
	address := testEntryIdentityAddress(t, ledger.AccountTypeCustomerFBO, "subaccount", ledger.Route{
		Currency: currencyx.Code("USD"),
	})
	identityKey, _ := ledger.EntryIdentityParts{
		CollectionSource: &collectionSource,
		SourceChargeID:   &sourceChargeID,
		SpendChargeID:    &spendChargeID,
	}.Text()

	t.Run("accepts canonical identity", func(t *testing.T) {
		entry := validationEntryInput{
			address:        address,
			amount:         alpacadecimal.NewFromInt(1),
			identityKey:    string(identityKey),
			sourceChargeID: &sourceChargeID,
			spendChargeID:  &spendChargeID,
		}

		require.NoError(t, ledger.ValidateEntryInput(t.Context(), entry))
	})

	t.Run("accepts current schema without charge provenance", func(t *testing.T) {
		entry := validationEntryInput{
			address:     address,
			amount:      alpacadecimal.NewFromInt(1),
			identityKey: "collection-source:0",
		}

		require.NoError(t, ledger.ValidateEntryInput(t.Context(), entry))
	})

	t.Run("rejects provenance without v2 identity", func(t *testing.T) {
		entry := validationEntryInput{
			address:        address,
			amount:         alpacadecimal.NewFromInt(1),
			identityKey:    "collection-source:0",
			sourceChargeID: &sourceChargeID,
		}

		err := ledger.ValidateEntryInput(t.Context(), entry)
		require.Error(t, err)
		require.ErrorContains(t, ledger.ValidateEntryIdentityKey(entry), "identity_key version must be 2")
	})

	t.Run("rejects mismatched source charge", func(t *testing.T) {
		otherSourceChargeID := "01JCDEFGH0123456789ABCDEFG"
		entry := validationEntryInput{
			address:        address,
			amount:         alpacadecimal.NewFromInt(1),
			identityKey:    string(identityKey),
			sourceChargeID: &otherSourceChargeID,
			spendChargeID:  &spendChargeID,
		}

		err := ledger.ValidateEntryInput(t.Context(), entry)
		require.Error(t, err)
		require.ErrorContains(t, ledger.ValidateEntryIdentityKey(entry), "source_charge_id does not match identity_key")
	})

	t.Run("rejects legacy schema with charge provenance", func(t *testing.T) {
		entry := validationEntryInput{
			address:        address,
			amount:         alpacadecimal.NewFromInt(1),
			identityKey:    "collection-source:0",
			schemaVersion:  ledger.EntrySchemaVersionLegacy,
			sourceChargeID: &sourceChargeID,
		}

		err := ledger.ValidateEntryInput(t.Context(), entry)
		require.Error(t, err)
		require.ErrorContains(t, ledger.ValidateEntryIdentityKey(entry), "schema_version 1 cannot contain charge provenance")
	})

	t.Run("rejects legacy schema with v2 identity", func(t *testing.T) {
		entry := validationEntryInput{
			address:       address,
			amount:        alpacadecimal.NewFromInt(1),
			identityKey:   string(identityKey),
			schemaVersion: ledger.EntrySchemaVersionLegacy,
		}

		err := ledger.ValidateEntryInput(t.Context(), entry)
		require.Error(t, err)
		require.ErrorContains(t, ledger.ValidateEntryIdentityKey(entry), "identity_key version 2 requires schema_version 2")
	})

	t.Run("rejects unsupported schema version", func(t *testing.T) {
		entry := validationEntryInput{
			address:       address,
			amount:        alpacadecimal.NewFromInt(1),
			identityKey:   "collection-source:0",
			schemaVersion: ledger.EntrySchemaVersion(99),
		}

		err := ledger.ValidateEntryInput(t.Context(), entry)
		require.Error(t, err)
		require.ErrorContains(t, ledger.ValidateEntryIdentityKey(entry), "unsupported schema_version 99")
	})

	t.Run("rejects non-canonical identity", func(t *testing.T) {
		entry := validationEntryInput{
			address:        address,
			amount:         alpacadecimal.NewFromInt(1),
			identityKey:    "entry-identity:v2:0||01JABCDEF0123456789ABCDEFG|01JBCDEFG0123456789ABCDEFG|extra",
			sourceChargeID: &sourceChargeID,
			spendChargeID:  &spendChargeID,
		}

		err := ledger.ValidateEntryInput(t.Context(), entry)
		require.Error(t, err)
		require.ErrorContains(t, ledger.ValidateEntryIdentityKey(entry), "invalid ledger entry identity key format")
	})
}

type validationEntryInput struct {
	address        ledger.PostingAddress
	amount         alpacadecimal.Decimal
	identityKey    string
	schemaVersion  ledger.EntrySchemaVersion
	sourceChargeID *string
	spendChargeID  *string
}

func (e validationEntryInput) PostingAddress() ledger.PostingAddress {
	return e.address
}

func (e validationEntryInput) Amount() alpacadecimal.Decimal {
	return e.amount
}

func (e validationEntryInput) IdentityKey() string {
	return e.identityKey
}

func (e validationEntryInput) SchemaVersion() ledger.EntrySchemaVersion {
	if e.schemaVersion == 0 {
		return ledger.EntrySchemaVersionCurrent
	}

	return e.schemaVersion
}

func (e validationEntryInput) SourceChargeID() *string {
	return e.sourceChargeID
}

func (e validationEntryInput) SpendChargeID() *string {
	return e.spendChargeID
}

func (e validationEntryInput) Annotations() models.Annotations {
	return nil
}

func testEntryIdentityAddress(t *testing.T, accountType ledger.AccountType, subAccountID string, route ledger.Route) ledger.PostingAddress {
	t.Helper()

	key, err := ledger.BuildRoutingKey(route)
	require.NoError(t, err)

	addr, err := ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
		SubAccountID: subAccountID,
		AccountType:  accountType,
		Route:        route,
		RouteID:      "route-" + subAccountID + "-" + time.Now().UTC().Format("150405.000000000"),
		RoutingKey:   key,
	})
	require.NoError(t, err)

	return addr
}
