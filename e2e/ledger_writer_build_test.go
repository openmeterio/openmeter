package e2e

import (
	"net/http"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestLedgerWriterBuild(t *testing.T) {
	c := newV3Client(t)
	pool := initE2EPostgresPool(t)

	// Given a customer on the running server.
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key: uniqueKey("writer_build"), Name: "Ledger writer build", Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)

	// When the server issues a promotional credit, it commits a ledger group.
	grant, err := c.Customers.Credits.Grants.Create(t.Context(), customer.ID, v3sdk.CreateCreditGrantRequest{
		Name: "writer build", Currency: "USD", Amount: "1", FundingMethod: v3sdk.CreditFundingMethodNone,
	})
	c.requireStatus(http.StatusCreated, err)

	// Then the persisted issuance group identifies the server build that wrote it.
	var annotations map[string]any
	err = pool.QueryRow(t.Context(), `
		SELECT annotations
		FROM ledger_transaction_groups
		WHERE annotations->>'ledger.charge.id' = $1
	`, grant.ID).Scan(&annotations)
	require.NoError(t, err)
	require.Regexp(t, "^[0-9a-f]{40}$", annotations[ledger.AnnotationWriterCommitSHA])
	if sha := os.Getenv("OPENMETER_E2E_EXPECTED_COMMIT_SHA"); sha != "" {
		require.Equal(t, sha, annotations[ledger.AnnotationWriterCommitSHA])
	}

	if version, ok := os.LookupEnv("OPENMETER_E2E_EXPECTED_VERSION"); ok {
		if version == "" {
			require.NotContains(t, annotations, ledger.AnnotationWriterVersion)
		} else {
			require.Equal(t, version, annotations[ledger.AnnotationWriterVersion])
		}
	}
}
