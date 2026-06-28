package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestNewCustomerHasLedgerAccounts(t *testing.T) {
	client := initClient(t)
	pool := initE2EPostgresPool(t)

	suffix := strings.ToLower(ulid.Make().String())
	customer := CreateCustomerWithSubject(t, client, "ledger-accounts-"+suffix, "ledger-accounts-subject-"+suffix)

	namespace := getCustomerNamespace(t, pool, customer.Id)
	customerMappings := getCustomerAccountMappings(t, pool, namespace, customer.Id)

	require.Len(t, customerMappings, len(ledger.CustomerAccountTypes), "unexpected customer account mapping count")
	for _, accountType := range ledger.CustomerAccountTypes {
		accountID, ok := customerMappings[accountType]
		require.True(t, ok, "missing customer mapping for type=%s", accountType)
		require.NotEmpty(t, accountID, "empty customer account id for type=%s", accountType)
	}

	businessAccounts := getBusinessAccountIDs(t, pool, namespace)
	require.Len(t, businessAccounts, len(ledger.BusinessAccountTypes))
	for _, accountType := range ledger.BusinessAccountTypes {
		accountID, ok := businessAccounts[accountType]
		require.True(t, ok, "missing business account mapping for type=%s", accountType)
		require.NotEmpty(t, accountID, "empty business account id for type=%s", accountType)
	}
}

func initE2EPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("OPENMETER_E2E_POSTGRES_URL")
	if dsn == "" {
		address := os.Getenv("OPENMETER_ADDRESS")
		if !strings.Contains(address, "localhost:38888") && !strings.Contains(address, "127.0.0.1:38888") {
			t.Skipf("ledger account e2e requires OPENMETER_E2E_POSTGRES_URL or local compose stack at localhost:38888, got %q", address)
		}

		dsn = "postgres://postgres:postgres@127.0.0.1:35432/postgres?sslmode=disable"
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(t.Context()))

	return pool
}

func getCustomerNamespace(t *testing.T, pool *pgxpool.Pool, customerID string) string {
	t.Helper()

	var namespace string
	err := pool.QueryRow(
		t.Context(),
		`SELECT namespace FROM customers WHERE id = $1`,
		customerID,
	).Scan(&namespace)
	require.NoError(t, err)

	return namespace
}

func getCustomerAccountMappings(t *testing.T, pool *pgxpool.Pool, namespace string, customerID string) map[ledger.AccountType]string {
	t.Helper()

	rows, err := pool.Query(
		t.Context(),
		`
SELECT lca.account_type, lca.account_id, la.id
FROM ledger_customer_accounts lca
JOIN ledger_accounts la
  ON la.namespace = lca.namespace
 AND la.id = lca.account_id
 AND la.account_type = lca.account_type
WHERE lca.namespace = $1
  AND lca.customer_id = $2
`,
		namespace,
		customerID,
	)
	require.NoError(t, err)
	defer rows.Close()

	result := map[ledger.AccountType]string{}
	for rows.Next() {
		var accountType string
		var accountID string
		var linkedAccountID string

		require.NoError(t, rows.Scan(&accountType, &accountID, &linkedAccountID))
		require.Equal(t, accountID, linkedAccountID, "customer mapping account id must match linked ledger account")

		typed := ledger.AccountType(accountType)
		require.Empty(t, typed.Validate(), "invalid customer account type returned: %s", accountType)
		require.NotContains(t, result, typed, "duplicate customer mapping for type=%s", accountType)

		result[typed] = accountID
	}

	require.NoError(t, rows.Err())

	return result
}

func getBusinessAccountIDs(t *testing.T, pool *pgxpool.Pool, namespace string) map[ledger.AccountType]string {
	t.Helper()

	accountTypes := make([]string, 0, len(ledger.BusinessAccountTypes))
	for _, accountType := range ledger.BusinessAccountTypes {
		accountTypes = append(accountTypes, string(accountType))
	}

	rows, err := pool.Query(
		t.Context(),
		`
SELECT account_type, id
FROM ledger_accounts
WHERE namespace = $1
  AND account_type = ANY($2)
`,
		namespace,
		accountTypes,
	)
	require.NoError(t, err)
	defer rows.Close()

	result := make(map[ledger.AccountType]string, len(ledger.BusinessAccountTypes))
	for rows.Next() {
		var accountType string
		var accountID string

		require.NoError(t, rows.Scan(&accountType, &accountID))

		typed := ledger.AccountType(accountType)
		require.Empty(t, typed.Validate(), "invalid business account type returned: %s", accountType)
		require.NotContains(t, result, typed, "duplicate business account type returned: %s", accountType)

		result[typed] = accountID
	}

	require.NoError(t, rows.Err())

	return result
}
