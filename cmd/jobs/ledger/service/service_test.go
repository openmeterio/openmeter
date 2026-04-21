package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func TestRunDryRun(t *testing.T) {
	now := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	provisioner := newFakeAccountProvisioner()
	provisioner.missingBusiness["ns-a"] = true
	provisioner.missingCustomer["ns-a/customer-1"] = true
	provisioner.missingCustomer["ns-a/customer-2"] = true

	svc, err := NewService(Config{
		CustomerLister: fakeCustomerLister{
			customersByNamespace: map[string][]customer.Customer{
				"ns-a": {
					newCustomer("ns-a", "customer-1", now.Add(-2*time.Hour)),
					newCustomer("ns-a", "customer-2", now.Add(-1*time.Hour)),
				},
			},
		},
		AccountProvisioner: provisioner,
	})
	require.NoError(t, err)

	out, err := svc.Run(t.Context(), RunInput{
		Namespace: "ns-a",
		DryRun:    true,
	})
	require.NoError(t, err)

	ns := out.Result
	require.Equal(t, 1, ns.BusinessWouldProvision)
	require.Equal(t, 2, ns.CustomersWouldProvision)
	require.Equal(t, 2, ns.CustomersScanned)
	require.Zero(t, ns.CustomersProvisioned)
	require.Zero(t, ns.BusinessProvisioned)

	require.Equal(t, 0, ns.FailureCount)
}

func TestRunCursorsThroughMultiplePages(t *testing.T) {
	now := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)

	provisioner := newFakeAccountProvisioner()
	provisioner.missingBusiness["ns-a"] = true
	provisioner.missingCustomer["ns-a/customer-1"] = true
	provisioner.missingCustomer["ns-a/customer-2"] = true
	provisioner.missingCustomer["ns-a/customer-3"] = true
	provisioner.missingCustomer["ns-a/customer-4"] = true
	provisioner.missingCustomer["ns-a/customer-5"] = true

	svc, err := NewService(Config{
		CustomerLister: fakeCustomerLister{
			customersByNamespace: map[string][]customer.Customer{
				"ns-a": {
					newCustomer("ns-a", "customer-1", now.Add(-5*time.Hour)),
					newCustomer("ns-a", "customer-2", now.Add(-4*time.Hour)),
					newCustomer("ns-a", "customer-3", now.Add(-3*time.Hour)),
					newCustomer("ns-a", "customer-4", now.Add(-2*time.Hour)),
					newCustomer("ns-a", "customer-5", now.Add(-1*time.Hour)),
				},
			},
		},
		AccountProvisioner: provisioner,
	})
	require.NoError(t, err)

	out, err := svc.Run(t.Context(), RunInput{
		Namespace:        "ns-a",
		DryRun:           true,
		CustomerPageSize: 2,
	})
	require.NoError(t, err)

	ns := out.Result
	require.Equal(t, 5, ns.CustomersScanned)
	require.Equal(t, 5, ns.CustomersWouldProvision)
	require.Equal(t, 1, ns.BusinessWouldProvision)
	require.Equal(t, 0, ns.FailureCount)
}

func TestRunProvisionWithCreatedBeforeCutoff(t *testing.T) {
	now := time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-30 * time.Minute)

	provisioner := newFakeAccountProvisioner()
	provisioner.missingBusiness["ns-a"] = true
	provisioner.missingCustomer["ns-a/customer-old"] = true
	provisioner.missingCustomer["ns-a/customer-recent"] = true

	svc, err := NewService(Config{
		CustomerLister: fakeCustomerLister{
			customersByNamespace: map[string][]customer.Customer{
				"ns-a": {
					newCustomer("ns-a", "customer-old", now.Add(-2*time.Hour)),
					newCustomer("ns-a", "customer-recent", now.Add(-10*time.Minute)),
				},
			},
		},
		AccountProvisioner: provisioner,
	})
	require.NoError(t, err)

	out, err := svc.Run(t.Context(), RunInput{
		Namespace:     "ns-a",
		CreatedBefore: &cutoff,
	})
	require.NoError(t, err)

	require.Equal(t, []customer.CustomerID{{Namespace: "ns-a", ID: "customer-old"}}, provisioner.createdCustomers)
	require.Equal(t, []string{"ns-a"}, provisioner.ensuredBusiness)

	ns := out.Result
	require.Equal(t, 1, ns.CustomersScanned)
	require.Equal(t, 0, ns.CustomersSkippedRecent)
	require.Equal(t, 1, ns.CustomersProvisioned)
	require.Zero(t, ns.CustomersWouldProvision)
	require.Equal(t, 1, ns.BusinessProvisioned)
	require.Equal(t, 0, ns.FailureCount)
}

func TestRunContinueOnError(t *testing.T) {
	provisioner := newFakeAccountProvisioner()
	provisioner.missingBusiness["ns-a"] = true
	provisioner.customerGetErrors["ns-a/bad"] = errors.New("db unavailable")
	provisioner.missingCustomer["ns-a/good"] = true

	svc, err := NewService(Config{
		CustomerLister: fakeCustomerLister{
			customersByNamespace: map[string][]customer.Customer{
				"ns-a": {
					newCustomer("ns-a", "bad", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
					newCustomer("ns-a", "good", time.Date(2026, 1, 2, 10, 1, 0, 0, time.UTC)),
				},
			},
		},
		AccountProvisioner: provisioner,
	})
	require.NoError(t, err)

	out, err := svc.Run(t.Context(), RunInput{
		Namespace:       "ns-a",
		ContinueOnError: true,
	})
	require.NoError(t, err)

	ns := out.Result
	require.Equal(t, 1, ns.CustomersProvisioned)
	require.Equal(t, 1, ns.FailureCount)
}

type fakeCustomerLister struct {
	customersByNamespace map[string][]customer.Customer
}

func (f fakeCustomerLister) ListCustomers(_ context.Context, input ListCustomersInput) (ListCustomersResult, error) {
	if err := input.Validate(); err != nil {
		return ListCustomersResult{}, err
	}

	items := f.customersByNamespace[input.Namespace]
	filtered := make([]customer.Customer, 0, len(items))
	for _, item := range items {
		if input.CreatedBefore != nil && !item.CreatedAt.Before(*input.CreatedBefore) {
			continue
		}
		filtered = append(filtered, item)
	}

	slices.SortFunc(filtered, func(a customer.Customer, b customer.Customer) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return stringsCompare(a.ID, b.ID)
	})

	start := 0
	if input.Cursor != nil {
		for idx, item := range filtered {
			if item.CreatedAt.After(input.Cursor.Time) {
				start = idx
				break
			}
			if item.CreatedAt.Equal(input.Cursor.Time) && stringsCompare(item.ID, input.Cursor.ID) > 0 {
				start = idx
				break
			}
			start = idx + 1
		}
	}

	if start >= len(filtered) {
		return ListCustomersResult{
			Items:      []CustomerListItem{},
			NextCursor: nil,
		}, nil
	}

	end := start + input.PageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	paged := make([]CustomerListItem, 0, end-start)
	for _, item := range filtered[start:end] {
		paged = append(paged, CustomerListItem{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
		})
	}

	var nextCursor *paginationv2.Cursor
	if len(paged) > 0 {
		last := paged[len(paged)-1]
		nextCursor = &paginationv2.Cursor{
			Time: last.CreatedAt,
			ID:   last.ID,
		}
	}

	return ListCustomersResult{
		Items:      paged,
		NextCursor: nextCursor,
	}, nil
}

func stringsCompare(a string, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}

	return 0
}

type fakeAccountProvisioner struct {
	missingBusiness map[string]bool
	missingCustomer map[string]bool

	customerGetErrors map[string]error

	ensuredBusiness  []string
	createdCustomers []customer.CustomerID
}

func newFakeAccountProvisioner() *fakeAccountProvisioner {
	return &fakeAccountProvisioner{
		missingBusiness:   make(map[string]bool),
		missingCustomer:   make(map[string]bool),
		customerGetErrors: map[string]error{},
	}
}

func (f *fakeAccountProvisioner) customerKey(id customer.CustomerID) string {
	return fmt.Sprintf("%s/%s", id.Namespace, id.ID)
}

func (f *fakeAccountProvisioner) GetCustomerAccounts(_ context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	key := f.customerKey(customerID)

	if err, ok := f.customerGetErrors[key]; ok {
		return ledger.CustomerAccounts{}, err
	}

	if f.missingCustomer[key] {
		return ledger.CustomerAccounts{}, ledger.ErrCustomerAccountMissing.WithAttrs(models.Attributes{
			"namespace":   customerID.Namespace,
			"customer_id": customerID.ID,
		})
	}

	return ledger.CustomerAccounts{}, nil
}

func (f *fakeAccountProvisioner) CreateCustomerAccounts(_ context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	key := f.customerKey(customerID)
	f.createdCustomers = append(f.createdCustomers, customerID)
	f.missingCustomer[key] = false

	return ledger.CustomerAccounts{}, nil
}

func (f *fakeAccountProvisioner) GetBusinessAccounts(_ context.Context, namespace string) (ledger.BusinessAccounts, error) {
	if f.missingBusiness[namespace] {
		return ledger.BusinessAccounts{}, ledger.ErrBusinessAccountMissing.WithAttrs(models.Attributes{
			"namespace": namespace,
		})
	}

	return ledger.BusinessAccounts{}, nil
}

func (f *fakeAccountProvisioner) EnsureBusinessAccounts(_ context.Context, namespace string) (ledger.BusinessAccounts, error) {
	f.ensuredBusiness = append(f.ensuredBusiness, namespace)
	f.missingBusiness[namespace] = false

	return ledger.BusinessAccounts{}, nil
}

func newCustomer(namespace string, id string, createdAt time.Time) customer.Customer {
	return customer.Customer{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:        id,
			Namespace: namespace,
			Name:      id,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}),
		Key: nil,
		UsageAttribution: &customer.CustomerUsageAttribution{
			SubjectKeys: []string{"sub-" + id},
		},
	}
}
