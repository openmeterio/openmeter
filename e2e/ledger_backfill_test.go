package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestLedgerBackfillAccountsJob(t *testing.T) {
	client := initClient(t)
	ensureLocalComposeBackfillSupport(t)

	pool := initE2EPostgresPool(t)

	suffix := strings.ToLower(ulid.Make().String())
	customerA := CreateCustomerWithSubject(t, client, "ledger-backfill-"+suffix+"-a", "ledger-backfill-subject-"+suffix+"-a")
	customerB := CreateCustomerWithSubject(t, client, "ledger-backfill-"+suffix+"-b", "ledger-backfill-subject-"+suffix+"-b")

	customerIDs := []string{customerA.Id, customerB.Id}
	namespace := getCustomerNamespace(t, pool, customerA.Id)
	require.Equal(t, namespace, getCustomerNamespace(t, pool, customerB.Id))

	before := getCustomerAccountMappingCounts(t, pool, namespace, customerIDs)
	for _, customerID := range customerIDs {
		require.Zero(t, before[customerID], "expected no pre-provisioned customer account mappings for %s", customerID)
	}

	output := runJobsBackfillAccounts(t, "--customer-page-size", "2")
	t.Logf("jobs output:\n%s", output)

	after := getCustomerAccountMappingCounts(t, pool, namespace, customerIDs)
	for _, customerID := range customerIDs {
		require.Equal(t, len(ledger.CustomerAccountTypes), after[customerID], "unexpected customer account mapping count for %s", customerID)
	}

	customerMappings := getCustomerAccountMappings(t, pool, namespace, customerIDs)
	for _, customerID := range customerIDs {
		require.Len(t, customerMappings[customerID], len(ledger.CustomerAccountTypes), "unexpected customer account mapping detail count for %s", customerID)
		for _, accountType := range ledger.CustomerAccountTypes {
			accountID, ok := customerMappings[customerID][accountType]
			require.True(t, ok, "missing customer mapping for %s type=%s", customerID, accountType)
			require.NotEmpty(t, accountID, "empty account id for customer=%s type=%s", customerID, accountType)
		}
	}

	businessAccounts := getBusinessAccountIDs(t, pool, namespace)
	require.Len(t, businessAccounts, len(ledger.BusinessAccountTypes))
	for _, accountType := range ledger.BusinessAccountTypes {
		accountID, ok := businessAccounts[accountType]
		require.True(t, ok, "missing business account mapping for type=%s", accountType)
		require.NotEmpty(t, accountID, "empty business account id for type=%s", accountType)
	}
}

func ensureLocalComposeBackfillSupport(t *testing.T) {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if !strings.Contains(address, "localhost:38888") && !strings.Contains(address, "127.0.0.1:38888") {
		t.Skipf("ledger backfill e2e requires local compose stack at localhost:38888, got %q", address)
	}

	cmd := exec.CommandContext(t.Context(), "docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		t.Skipf("docker compose is required for ledger backfill e2e: %v", err)
	}
}

func initE2EPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("OPENMETER_E2E_POSTGRES_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@127.0.0.1:35432/postgres?sslmode=disable"
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(t.Context()))

	return pool
}

func runJobsBackfillAccounts(t *testing.T, args ...string) string {
	t.Helper()

	base := []string{
		"compose",
		"-f", "docker-compose.infra.yaml",
		"-f", "docker-compose.debug-ports.yaml",
		"-f", "docker-compose.openmeter.yaml",
		"-f", "docker-compose.openmeter-local.yaml",
		"exec",
		"-T",
		"openmeter",
		"openmeter-jobs",
		"--config", "/etc/openmeter/config.yaml",
		"ledger",
		"backfill-accounts",
	}
	base = append(base, args...)

	cmd := exec.CommandContext(t.Context(), "docker", base...)
	cmd.Dir = "."

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	require.NoErrorf(t, cmd.Run(), "ledger backfill job command failed:\n%s", out.String())

	return out.String()
}

func getCustomerAccountMappingCounts(t *testing.T, pool *pgxpool.Pool, namespace string, customerIDs []string) map[string]int {
	t.Helper()

	query := `
SELECT customer_id, COUNT(*)
FROM ledger_customer_accounts
WHERE namespace = $1
  AND customer_id = ANY($2)
GROUP BY customer_id
`

	rows, err := pool.Query(t.Context(), query, namespace, customerIDs)
	require.NoError(t, err)
	defer rows.Close()

	counts := make(map[string]int, len(customerIDs))
	for _, customerID := range customerIDs {
		counts[customerID] = 0
	}

	for rows.Next() {
		var customerID string
		var count int

		require.NoError(t, rows.Scan(&customerID, &count))

		counts[customerID] = count
	}

	require.NoError(t, rows.Err())

	return counts
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

func getCustomerAccountMappings(t *testing.T, pool *pgxpool.Pool, namespace string, customerIDs []string) map[string]map[ledger.AccountType]string {
	t.Helper()

	rows, err := pool.Query(
		t.Context(),
		`
SELECT lca.customer_id, lca.account_type, lca.account_id, la.id
FROM ledger_customer_accounts lca
JOIN ledger_accounts la
  ON la.namespace = lca.namespace
 AND la.id = lca.account_id
 AND la.account_type = lca.account_type
WHERE lca.namespace = $1
  AND lca.customer_id = ANY($2)
`,
		namespace,
		customerIDs,
	)
	require.NoError(t, err)
	defer rows.Close()

	result := make(map[string]map[ledger.AccountType]string, len(customerIDs))
	for _, customerID := range customerIDs {
		result[customerID] = map[ledger.AccountType]string{}
	}

	for rows.Next() {
		var customerID string
		var accountType string
		var accountID string
		var linkedAccountID string

		require.NoError(t, rows.Scan(&customerID, &accountType, &accountID, &linkedAccountID))
		require.Equal(t, accountID, linkedAccountID, "customer mapping account id must match linked ledger account")

		typed := ledger.AccountType(accountType)
		require.Empty(t, typed.Validate(), "invalid customer account type returned: %s", accountType)
		require.NotContains(t, result[customerID], typed, "duplicate mapping for customer=%s type=%s", customerID, accountType)

		result[customerID][typed] = accountID
	}

	require.NoError(t, rows.Err())

	return result
}
