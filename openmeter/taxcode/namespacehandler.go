package taxcode

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SeedEntry defines a tax code that should be provisioned for every namespace.
type SeedEntry struct {
	Key                string
	Name               string
	Description        *string
	AppMappings        TaxCodeAppMappings
	DefaultInvoicing   bool
	DefaultCreditGrant bool
}

// NamespaceHandlerConfig holds the dependencies for the taxcode namespace handler.
type NamespaceHandlerConfig struct {
	Logger             *slog.Logger
	Service            Service
	Seeds              []SeedEntry
	TransactionManager transaction.Creator
}

func (c NamespaceHandlerConfig) validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Service == nil {
		errs = append(errs, errors.New("service is required"))
	}

	if c.TransactionManager == nil {
		errs = append(errs, errors.New("transaction manager is required"))
	}

	if len(c.Seeds) == 0 {
		errs = append(errs, errors.New("at least one seed entry is required"))
	} else {
		invoicingCount := lo.CountBy(c.Seeds, func(s SeedEntry) bool { return s.DefaultInvoicing })
		creditGrantCount := lo.CountBy(c.Seeds, func(s SeedEntry) bool { return s.DefaultCreditGrant })
		if invoicingCount != 1 {
			errs = append(errs, fmt.Errorf("exactly one seed must have DefaultInvoicing=true, got %d", invoicingCount))
		}
		if creditGrantCount != 1 {
			errs = append(errs, fmt.Errorf("exactly one seed must have DefaultCreditGrant=true, got %d", creditGrantCount))
		}
	}

	return errors.Join(errs...)
}

// NamespaceHandler implements namespace.Handler for the taxcode domain.
type NamespaceHandler struct {
	logger             *slog.Logger
	service            Service
	seeds              []SeedEntry
	transactionManager transaction.Creator
}

var _ namespace.Handler = (*NamespaceHandler)(nil)

// NewNamespaceHandler creates a *NamespaceHandler that seeds tax codes and org
// defaults when a new namespace is provisioned.
func NewNamespaceHandler(cfg NamespaceHandlerConfig) (*NamespaceHandler, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid namespace handler config: %w", err)
	}

	return &NamespaceHandler{
		logger:             cfg.Logger,
		service:            cfg.Service,
		seeds:              cfg.Seeds,
		transactionManager: cfg.TransactionManager,
	}, nil
}

// CreateNamespace provisions the configured seed tax codes and sets the per-namespace
// OrganizationDefaultTaxCodes. The operation is idempotent: pre-existing tax codes are
// left unchanged and a pre-existing org-defaults row skips the org-defaults upsert (seed
// creation always runs regardless).
// All seed creates and the org-defaults upsert run inside a single transaction.
func (h *NamespaceHandler) CreateNamespace(ctx context.Context, ns string) error {
	return transaction.RunWithNoValue(ctx, h.transactionManager, func(ctx context.Context) error {
		// List existing tax codes once; ensureTaxCode does map lookups against this set
		// and only re-lists on a concurrent-create conflict.
		listed, err := h.service.ListTaxCodes(ctx, ListTaxCodesInput{Namespace: ns})
		if err != nil {
			return fmt.Errorf("list tax codes: %w", err)
		}
		existingByKey := lo.SliceToMap(listed.Items, func(tc TaxCode) (string, TaxCode) {
			return tc.Key, tc
		})

		var invoicingID, creditGrantID string

		for _, seed := range h.seeds {
			id, err := h.ensureTaxCode(ctx, ns, seed, existingByKey)
			if err != nil {
				return fmt.Errorf("seed tax code %q: %w", seed.Key, err)
			}

			if seed.DefaultInvoicing {
				invoicingID = id
			}

			if seed.DefaultCreditGrant {
				creditGrantID = id
			}
		}

		// Idempotency check: if org defaults already exist, skip upsert.
		_, err = h.service.GetOrganizationDefaultTaxCodes(ctx, GetOrganizationDefaultTaxCodesInput{
			Namespace: ns,
		})
		if err != nil && !IsOrganizationDefaultTaxCodesNotFoundError(err) {
			return fmt.Errorf("get organization default tax codes: %w", err)
		}

		if err == nil {
			// Already provisioned — nothing to do.
			return nil
		}

		if _, err := h.service.UpsertOrganizationDefaultTaxCodes(ctx, UpsertOrganizationDefaultTaxCodesInput{
			Namespace:            ns,
			InvoicingTaxCodeID:   invoicingID,
			CreditGrantTaxCodeID: creditGrantID,
		}); err != nil {
			return fmt.Errorf("upsert organization default tax codes: %w", err)
		}

		return nil
	})
}

// DeleteNamespace is a no-op; tax codes belong to the namespace and are cleaned up
// by the database cascade or a dedicated purge job.
func (h *NamespaceHandler) DeleteNamespace(_ context.Context, _ string) error {
	return nil
}

// ensureTaxCode returns the ID of the tax code identified by seed.Key in namespace ns.
// If it does not exist yet, it is created. Pre-existing codes are never mutated.
// existingByKey is the pre-fetched index built by CreateNamespace; the conflict path
// re-lists from the service to reconcile concurrent inserts that the index missed.
func (h *NamespaceHandler) ensureTaxCode(ctx context.Context, ns string, seed SeedEntry, existingByKey map[string]TaxCode) (string, error) {
	if existing, found := existingByKey[seed.Key]; found {
		return existing.ID, nil
	}

	// Not found — create it.
	created, createErr := h.service.CreateTaxCode(ctx, CreateTaxCodeInput{
		Namespace:   ns,
		Key:         seed.Key,
		Name:        seed.Name,
		Description: seed.Description,
		AppMappings: seed.AppMappings,
		Annotations: models.Annotations{
			AnnotationKeyManagedBy: AnnotationValueManagedBySystem,
		},
	})
	if createErr != nil {
		// Another goroutine may have created it concurrently; re-fetch by listing.
		if models.IsGenericConflictError(createErr) {
			result2, listErr := h.service.ListTaxCodes(ctx, ListTaxCodesInput{
				Namespace: ns,
			})
			if listErr != nil {
				return "", fmt.Errorf("list tax codes after conflict: %w", listErr)
			}

			concurrent, ok := lo.Find(result2.Items, func(tc TaxCode) bool {
				return tc.Key == seed.Key
			})
			if ok {
				return concurrent.ID, nil
			}

			return "", fmt.Errorf("tax code with key %q: conflict reported by create but key not found after re-list: %w", seed.Key, createErr)
		}

		return "", fmt.Errorf("create tax code: %w", createErr)
	}

	return created.ID, nil
}
