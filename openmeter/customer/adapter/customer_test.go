package adapter_test

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	appcustominvoicingcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicingcustomer"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type fixture struct {
	customerID                      string
	subjectKeys                     []string
	appID                           string
	appCustomerRowID                int
	appStripeCustomerID             int
	customInvoicingAppID            string
	appCustomInvoicingCustomerRowID int
}

type testEnv struct {
	t       *testing.T
	db      *entdb.Client
	adapter customer.Adapter
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	testdb := testutils.InitPostgresDB(t)
	dbClient := testdb.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testdb.Close(t)
	})

	require.NoError(t, dbClient.Schema.Create(t.Context()), "schema migration must not fail")

	adapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err, "constructing customer adapter must not fail")

	return &testEnv{
		t:       t,
		db:      dbClient,
		adapter: adapter,
	}
}

// seed creates a Customer in the given namespace plus a full cascade chain:
// 2 CustomerSubjects, an App (Stripe), an AppStripe, 1 AppCustomer,
// 1 AppStripeCustomer, an App (CustomInvoicing), an AppCustomInvoicing, and
// 1 AppCustomInvoicingCustomer — all with deleted_at = NULL.
func (e *testEnv) seed(namespace string) fixture {
	e.t.Helper()
	ctx := e.t.Context()

	cust, err := e.db.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer-" + namespace).
		Save(ctx)
	require.NoError(e.t, err, "seeding customer must not fail")

	subjectKeys := []string{"subject-a-" + namespace, "subject-b-" + namespace}
	_, err = e.db.CustomerSubjects.CreateBulk(
		e.db.CustomerSubjects.Create().
			SetNamespace(namespace).
			SetCustomerID(cust.ID).
			SetSubjectKey(subjectKeys[0]),
		e.db.CustomerSubjects.Create().
			SetNamespace(namespace).
			SetCustomerID(cust.ID).
			SetSubjectKey(subjectKeys[1]),
	).Save(ctx)
	require.NoError(e.t, err, "seeding customer subjects must not fail")

	appRow, err := e.db.App.Create().
		SetNamespace(namespace).
		SetName("stripe").
		SetType(app.AppTypeStripe).
		SetStatus(app.AppStatusReady).
		Save(ctx)
	require.NoError(e.t, err, "seeding app must not fail")

	_, err = e.db.AppStripe.Create().
		SetID(appRow.ID).
		SetNamespace(namespace).
		SetStripeAccountID("acct_test_" + namespace).
		SetStripeLivemode(false).
		SetAPIKey("sk_test_" + namespace).
		SetMaskedAPIKey("sk_test_****").
		SetStripeWebhookID("we_test_" + namespace).
		SetWebhookSecret("whsec_" + namespace).
		Save(ctx)
	require.NoError(e.t, err, "seeding app stripe must not fail")

	appCust, err := e.db.AppCustomer.Create().
		SetNamespace(namespace).
		SetAppID(appRow.ID).
		SetCustomerID(cust.ID).
		Save(ctx)
	require.NoError(e.t, err, "seeding app customer must not fail")

	appStripeCust, err := e.db.AppStripeCustomer.Create().
		SetNamespace(namespace).
		SetAppID(appRow.ID).
		SetCustomerID(cust.ID).
		SetStripeCustomerID("cus_test_" + namespace).
		Save(ctx)
	require.NoError(e.t, err, "seeding app stripe customer must not fail")

	ciAppRow, err := e.db.App.Create().
		SetNamespace(namespace).
		SetName("custom-invoicing").
		SetType(app.AppTypeCustomInvoicing).
		SetStatus(app.AppStatusReady).
		Save(ctx)
	require.NoError(e.t, err, "seeding custom invoicing app must not fail")

	_, err = e.db.AppCustomInvoicing.Create().
		SetID(ciAppRow.ID).
		SetNamespace(namespace).
		Save(ctx)
	require.NoError(e.t, err, "seeding app custom invoicing must not fail")

	ciCust, err := e.db.AppCustomInvoicingCustomer.Create().
		SetNamespace(namespace).
		SetAppID(ciAppRow.ID).
		SetCustomerID(cust.ID).
		Save(ctx)
	require.NoError(e.t, err, "seeding app custom invoicing customer must not fail")

	return fixture{
		customerID:                      cust.ID,
		subjectKeys:                     subjectKeys,
		appID:                           appRow.ID,
		appCustomerRowID:                appCust.ID,
		appStripeCustomerID:             appStripeCust.ID,
		customInvoicingAppID:            ciAppRow.ID,
		appCustomInvoicingCustomerRowID: ciCust.ID,
	}
}

// freezeTime freezes the wall clock at a microsecond-truncated UTC instant so
// the value persists round-trips exactly through Postgres (which has microsecond
// precision). The unfreeze is registered with t.Cleanup.
func freezeTime(t *testing.T, at time.Time) time.Time {
	t.Helper()
	frozen := at.UTC().Truncate(time.Microsecond)
	clock.FreezeTime(frozen)
	t.Cleanup(clock.UnFreeze)
	return frozen
}

func TestDeleteCustomer(t *testing.T) {
	t.Run("Cascade_AllChildren", func(t *testing.T) {
		env := newTestEnv(t)
		ns := ulid.Make().String()
		fix := env.seed(ns)

		now := freezeTime(t, time.Now())

		err := env.adapter.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
			Namespace: ns,
			ID:        fix.customerID,
		})
		require.NoError(t, err, "delete must not fail")

		assertCustomerDeletedAt(t, env.db, ns, fix.customerID, now)
		assertSubjectsDeletedAt(t, env.db, ns, fix.customerID, fix.subjectKeys, now)
		assertAppCustomerDeletedAt(t, env.db, ns, fix.customerID, now)
		assertAppStripeCustomerDeletedAt(t, env.db, ns, fix.customerID, now)
		assertAppCustomInvoicingCustomerDeletedAt(t, env.db, ns, fix.customerID, now)
	})

	t.Run("NoChildren", func(t *testing.T) {
		env := newTestEnv(t)
		ns := ulid.Make().String()

		cust, err := env.db.Customer.Create().
			SetNamespace(ns).
			SetName("orphan").
			Save(t.Context())
		require.NoError(t, err)

		now := freezeTime(t, time.Now())

		err = env.adapter.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
			Namespace: ns,
			ID:        cust.ID,
		})
		require.NoError(t, err, "delete on customer without children must not fail")

		assertCustomerDeletedAt(t, env.db, ns, cust.ID, now)
	})

	t.Run("PreservesAlreadyDeletedChildren", func(t *testing.T) {
		env := newTestEnv(t)
		ns := ulid.Make().String()
		fix := env.seed(ns)

		// Pre-soft-delete one subject and the app_customer at t0; they must NOT be
		// overwritten when DeleteCustomer runs at t1 because the cascade filters by
		// `deleted_at IS NULL`.
		t0 := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
		ctx := t.Context()

		preDeletedSubject := fix.subjectKeys[0]
		_, err := env.db.CustomerSubjects.Update().
			Where(
				customersubjectsdb.Namespace(ns),
				customersubjectsdb.CustomerID(fix.customerID),
				customersubjectsdb.SubjectKey(preDeletedSubject),
			).
			SetDeletedAt(t0).
			Save(ctx)
		require.NoError(t, err)

		_, err = env.db.AppCustomer.Update().
			Where(
				appcustomerdb.Namespace(ns),
				appcustomerdb.CustomerID(fix.customerID),
			).
			SetDeletedAt(t0).
			Save(ctx)
		require.NoError(t, err)

		t1 := freezeTime(t, time.Now())

		err = env.adapter.DeleteCustomer(ctx, customer.DeleteCustomerInput{
			Namespace: ns,
			ID:        fix.customerID,
		})
		require.NoError(t, err)

		// Customer itself: deleted at t1.
		assertCustomerDeletedAt(t, env.db, ns, fix.customerID, t1)

		// Pre-deleted subject keeps t0; the other subject gets t1.
		subjects, err := env.db.CustomerSubjects.Query().
			Where(
				customersubjectsdb.Namespace(ns),
				customersubjectsdb.CustomerID(fix.customerID),
			).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, subjects, 2)
		for _, s := range subjects {
			require.NotNil(t, s.DeletedAt, "every subject must be soft-deleted after the cascade")
			if s.SubjectKey == preDeletedSubject {
				assert.Truef(t, s.DeletedAt.Equal(t0),
					"pre-deleted subject must keep t0=%s, got %s", t0, s.DeletedAt)
			} else {
				assert.Truef(t, s.DeletedAt.Equal(t1),
					"active subject must be deleted at t1=%s, got %s", t1, s.DeletedAt)
			}
		}

		// app_customer: pre-deleted, must keep t0.
		appCusts, err := env.db.AppCustomer.Query().
			Where(
				appcustomerdb.Namespace(ns),
				appcustomerdb.CustomerID(fix.customerID),
			).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, appCusts, 1)
		require.NotNil(t, appCusts[0].DeletedAt)
		assert.Truef(t, appCusts[0].DeletedAt.Equal(t0),
			"pre-deleted app_customer must keep t0=%s, got %s", t0, appCusts[0].DeletedAt)

		// app_stripe_customer: was active, must be deleted at t1.
		assertAppStripeCustomerDeletedAt(t, env.db, ns, fix.customerID, t1)

		// app_custom_invoicing_customer: was active, must be deleted at t1.
		assertAppCustomInvoicingCustomerDeletedAt(t, env.db, ns, fix.customerID, t1)
	})

	t.Run("NotFound", func(t *testing.T) {
		env := newTestEnv(t)
		ns := ulid.Make().String()

		err := env.adapter.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
			Namespace: ns,
			ID:        ulid.Make().String(),
		})

		var notFoundErr *models.GenericNotFoundError
		require.ErrorAs(t, err, &notFoundErr, "deleting an unknown customer must return GenericNotFoundError")
	})

	t.Run("AlreadyDeletedCustomer", func(t *testing.T) {
		env := newTestEnv(t)
		ns := ulid.Make().String()
		fix := env.seed(ns)

		_ = freezeTime(t, time.Now())

		require.NoError(t, env.adapter.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
			Namespace: ns,
			ID:        fix.customerID,
		}))

		err := env.adapter.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
			Namespace: ns,
			ID:        fix.customerID,
		})

		var notFoundErr *models.GenericNotFoundError
		require.ErrorAs(t, err, &notFoundErr, "deleting an already-deleted customer must return GenericNotFoundError")
	})

	t.Run("DifferentNamespaceIsolation", func(t *testing.T) {
		env := newTestEnv(t)
		nsA := ulid.Make().String()
		nsB := ulid.Make().String()
		fixA := env.seed(nsA)
		fixB := env.seed(nsB)

		now := freezeTime(t, time.Now())

		err := env.adapter.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
			Namespace: nsA,
			ID:        fixA.customerID,
		})
		require.NoError(t, err)

		// nsA: full cascade applied.
		assertCustomerDeletedAt(t, env.db, nsA, fixA.customerID, now)
		assertSubjectsDeletedAt(t, env.db, nsA, fixA.customerID, fixA.subjectKeys, now)
		assertAppCustomerDeletedAt(t, env.db, nsA, fixA.customerID, now)
		assertAppStripeCustomerDeletedAt(t, env.db, nsA, fixA.customerID, now)
		assertAppCustomInvoicingCustomerDeletedAt(t, env.db, nsA, fixA.customerID, now)

		// nsB: completely untouched.
		assertCustomerActive(t, env.db, nsB, fixB.customerID)
		assertSubjectsActive(t, env.db, nsB, fixB.customerID, fixB.subjectKeys)
		assertAppCustomerActive(t, env.db, nsB, fixB.customerID)
		assertAppStripeCustomerActive(t, env.db, nsB, fixB.customerID)
		assertAppCustomInvoicingCustomerActive(t, env.db, nsB, fixB.customerID)
	})
}

// --- assertion helpers ---

func assertCustomerDeletedAt(t *testing.T, db *entdb.Client, ns, id string, want time.Time) {
	t.Helper()
	c, err := db.Customer.Query().
		Where(customerdb.Namespace(ns), customerdb.ID(id)).
		Only(t.Context())
	require.NoError(t, err, "customer must exist")
	require.NotNil(t, c.DeletedAt, "customer.deleted_at must be set")
	assert.Truef(t, c.DeletedAt.Equal(want),
		"customer.deleted_at: want %s, got %s", want, c.DeletedAt)
}

func assertCustomerActive(t *testing.T, db *entdb.Client, ns, id string) {
	t.Helper()
	c, err := db.Customer.Query().
		Where(customerdb.Namespace(ns), customerdb.ID(id)).
		Only(t.Context())
	require.NoError(t, err)
	assert.Nil(t, c.DeletedAt, "customer must remain active")
}

func assertSubjectsDeletedAt(t *testing.T, db *entdb.Client, ns, customerID string, keys []string, want time.Time) {
	t.Helper()
	subjects, err := db.CustomerSubjects.Query().
		Where(
			customersubjectsdb.Namespace(ns),
			customersubjectsdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.Len(t, subjects, len(keys), "expected %d subjects", len(keys))
	for _, s := range subjects {
		require.NotNilf(t, s.DeletedAt, "subject %s deleted_at must be set", s.SubjectKey)
		assert.Truef(t, s.DeletedAt.Equal(want),
			"subject %s deleted_at: want %s, got %s", s.SubjectKey, want, s.DeletedAt)
	}
}

func assertSubjectsActive(t *testing.T, db *entdb.Client, ns, customerID string, keys []string) {
	t.Helper()
	subjects, err := db.CustomerSubjects.Query().
		Where(
			customersubjectsdb.Namespace(ns),
			customersubjectsdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.Len(t, subjects, len(keys))
	for _, s := range subjects {
		assert.Nilf(t, s.DeletedAt, "subject %s must remain active", s.SubjectKey)
	}
}

func assertAppCustomerDeletedAt(t *testing.T, db *entdb.Client, ns, customerID string, want time.Time) {
	t.Helper()
	rows, err := db.AppCustomer.Query().
		Where(
			appcustomerdb.Namespace(ns),
			appcustomerdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, rows, "expected at least one app_customer row")
	for _, r := range rows {
		require.NotNilf(t, r.DeletedAt, "app_customer %d deleted_at must be set", r.ID)
		assert.Truef(t, r.DeletedAt.Equal(want),
			"app_customer %d deleted_at: want %s, got %s", r.ID, want, r.DeletedAt)
	}
}

func assertAppCustomerActive(t *testing.T, db *entdb.Client, ns, customerID string) {
	t.Helper()
	rows, err := db.AppCustomer.Query().
		Where(
			appcustomerdb.Namespace(ns),
			appcustomerdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, rows, "expected at least one app_customer row")
	for _, r := range rows {
		assert.Nilf(t, r.DeletedAt, "app_customer %d must remain active", r.ID)
	}
}

func assertAppStripeCustomerDeletedAt(t *testing.T, db *entdb.Client, ns, customerID string, want time.Time) {
	t.Helper()
	rows, err := db.AppStripeCustomer.Query().
		Where(
			appstripecustomerdb.Namespace(ns),
			appstripecustomerdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, rows, "expected at least one app_stripe_customer row")
	for _, r := range rows {
		require.NotNilf(t, r.DeletedAt, "app_stripe_customer %d deleted_at must be set", r.ID)
		assert.Truef(t, r.DeletedAt.Equal(want),
			"app_stripe_customer %d deleted_at: want %s, got %s", r.ID, want, r.DeletedAt)
	}
}

func assertAppStripeCustomerActive(t *testing.T, db *entdb.Client, ns, customerID string) {
	t.Helper()
	rows, err := db.AppStripeCustomer.Query().
		Where(
			appstripecustomerdb.Namespace(ns),
			appstripecustomerdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, rows, "expected at least one app_stripe_customer row")
	for _, r := range rows {
		assert.Nilf(t, r.DeletedAt, "app_stripe_customer %d must remain active", r.ID)
	}
}

func assertAppCustomInvoicingCustomerDeletedAt(t *testing.T, db *entdb.Client, ns, customerID string, want time.Time) {
	t.Helper()
	rows, err := db.AppCustomInvoicingCustomer.Query().
		Where(
			appcustominvoicingcustomerdb.Namespace(ns),
			appcustominvoicingcustomerdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, rows, "expected at least one app_custom_invoicing_customer row")
	for _, r := range rows {
		require.NotNilf(t, r.DeletedAt, "app_custom_invoicing_customer %d deleted_at must be set", r.ID)
		assert.Truef(t, r.DeletedAt.Equal(want),
			"app_custom_invoicing_customer %d deleted_at: want %s, got %s", r.ID, want, r.DeletedAt)
	}
}

func assertAppCustomInvoicingCustomerActive(t *testing.T, db *entdb.Client, ns, customerID string) {
	t.Helper()
	rows, err := db.AppCustomInvoicingCustomer.Query().
		Where(
			appcustominvoicingcustomerdb.Namespace(ns),
			appcustominvoicingcustomerdb.CustomerID(customerID),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, rows, "expected at least one app_custom_invoicing_customer row")
	for _, r := range rows {
		assert.Nilf(t, r.DeletedAt, "app_custom_invoicing_customer %d must remain active", r.ID)
	}
}
