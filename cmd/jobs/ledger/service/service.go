package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const DefaultCustomerPageSize = 100

type customerLister interface {
	ListCustomers(ctx context.Context, input ListCustomersInput) (ListCustomersResult, error)
}

type ListCustomersInput struct {
	Namespace      string
	IncludeDeleted bool
	CreatedBefore  *time.Time
	PageSize       int
	Cursor         *paginationv2.Cursor
}

func (i ListCustomersInput) Validate() error {
	if i.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if i.PageSize <= 0 {
		return fmt.Errorf("page size must be greater than zero")
	}

	return nil
}

type CustomerListItem struct {
	ID        string
	CreatedAt time.Time
}

type ListCustomersResult struct {
	Items      []CustomerListItem
	NextCursor *paginationv2.Cursor
}

type accountProvisioner interface {
	CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error)
	GetCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error)
	EnsureBusinessAccounts(ctx context.Context, namespace string) (ledger.BusinessAccounts, error)
	GetBusinessAccounts(ctx context.Context, namespace string) (ledger.BusinessAccounts, error)
}

type Config struct {
	CustomerLister     customerLister
	AccountProvisioner accountProvisioner
	Logger             *slog.Logger
}

func (c Config) Validate() error {
	if c.CustomerLister == nil {
		return fmt.Errorf("customer lister is required")
	}

	if c.AccountProvisioner == nil {
		return fmt.Errorf("account provisioner is required")
	}

	return nil
}

type Service struct {
	customerLister     customerLister
	accountProvisioner accountProvisioner
	logger             *slog.Logger
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid backfill config: %w", err)
	}

	return &Service{
		customerLister:     cfg.CustomerLister,
		accountProvisioner: cfg.AccountProvisioner,
		logger:             cfg.Logger,
	}, nil
}

type RunInput struct {
	Namespace string

	DryRun          bool
	ContinueOnError bool

	IncludeDeletedCustomers bool
	CustomerPageSize        int
	CreatedBefore           *time.Time
}

func (i RunInput) Validate() error {
	if i.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if i.CustomerPageSize < 0 {
		return fmt.Errorf("customer page size cannot be negative")
	}

	return nil
}

func (i RunInput) normalized() RunInput {
	out := i

	if out.CustomerPageSize == 0 {
		out.CustomerPageSize = DefaultCustomerPageSize
	}

	if out.CreatedBefore != nil {
		t := out.CreatedBefore.UTC()
		out.CreatedBefore = &t
	}

	return out
}

type RunOutput struct {
	Result NamespaceResult
}

type NamespaceResult struct {
	Namespace string

	BusinessAlreadyProvisioned int
	BusinessWouldProvision     int
	BusinessProvisioned        int

	CustomersScanned            int
	CustomersSkippedRecent      int
	CustomersAlreadyProvisioned int
	CustomersWouldProvision     int
	CustomersProvisioned        int

	FailureCount int
}

func (s *Service) Run(ctx context.Context, in RunInput) (RunOutput, error) {
	if err := in.Validate(); err != nil {
		return RunOutput{}, fmt.Errorf("invalid backfill input: %w", err)
	}

	input := in.normalized()
	result, err := s.runNamespace(ctx, input, input.Namespace)
	output := RunOutput{Result: result}

	if err != nil {
		if input.ContinueOnError {
			return output, nil
		}

		return output, err
	}

	return output, nil
}

func (s *Service) runNamespace(ctx context.Context, input RunInput, namespace string) (NamespaceResult, error) {
	result := NamespaceResult{Namespace: namespace}

	if err := s.ensureBusinessAccounts(ctx, input, &result); err != nil {
		if !input.ContinueOnError {
			return result, err
		}
	}

	var cursor *paginationv2.Cursor
	completed := false

	for iter := 0; iter < paginationv2.MAX_SAFE_ITER; iter++ {
		res, err := s.customerLister.ListCustomers(ctx, ListCustomersInput{
			Namespace:      namespace,
			IncludeDeleted: input.IncludeDeletedCustomers,
			CreatedBefore:  input.CreatedBefore,
			PageSize:       input.CustomerPageSize,
			Cursor:         cursor,
		})
		if err != nil {
			failure := s.recordFailure(&result, "list_customers", "", err)
			if input.ContinueOnError {
				return result, nil
			}

			return result, failure
		}

		if len(res.Items) == 0 {
			completed = true
			break
		}

		for _, item := range res.Items {
			result.CustomersScanned++

			if err := s.ensureCustomerAccounts(ctx, input, &result, customer.CustomerID{Namespace: namespace, ID: item.ID}); err != nil {
				if !input.ContinueOnError {
					return result, err
				}
			}
		}

		if res.NextCursor == nil {
			completed = true
			break
		}

		cursor = res.NextCursor
	}

	if !completed {
		failure := s.recordFailure(&result, "paginate_customers", "", fmt.Errorf("max safe iter reached: %d", paginationv2.MAX_SAFE_ITER))
		if input.ContinueOnError {
			return result, nil
		}

		return result, failure
	}

	return result, nil
}

func (s *Service) ensureBusinessAccounts(ctx context.Context, input RunInput, result *NamespaceResult) error {
	_, err := s.accountProvisioner.GetBusinessAccounts(ctx, result.Namespace)
	if err == nil {
		result.BusinessAlreadyProvisioned++
		return nil
	}

	if !hasValidationIssueCode(err, ledger.ErrCodeBusinessAccountMissing) {
		failure := s.recordFailure(result, "get_business_accounts", "", err)
		return failure
	}

	if input.DryRun {
		result.BusinessWouldProvision++
		return nil
	}

	_, err = s.accountProvisioner.EnsureBusinessAccounts(ctx, result.Namespace)
	if err != nil {
		failure := s.recordFailure(result, "ensure_business_accounts", "", err)
		return failure
	}

	result.BusinessProvisioned++

	return nil
}

func (s *Service) ensureCustomerAccounts(ctx context.Context, input RunInput, result *NamespaceResult, customerID customer.CustomerID) error {
	_, err := s.accountProvisioner.GetCustomerAccounts(ctx, customerID)
	if err == nil {
		result.CustomersAlreadyProvisioned++
		return nil
	}

	if !hasValidationIssueCode(err, ledger.ErrCodeCustomerAccountMissing) {
		failure := s.recordFailure(result, "get_customer_accounts", customerID.ID, err)
		return failure
	}

	if input.DryRun {
		result.CustomersWouldProvision++
		return nil
	}

	_, err = s.accountProvisioner.CreateCustomerAccounts(ctx, customerID)
	if err != nil {
		failure := s.recordFailure(result, "create_customer_accounts", customerID.ID, err)
		return failure
	}

	result.CustomersProvisioned++

	return nil
}

func hasValidationIssueCode(err error, code models.ErrorCode) bool {
	issues, issueErr := models.AsValidationIssues(err)
	if issueErr != nil {
		return false
	}

	for _, issue := range issues {
		if issue.Code() == code {
			return true
		}
	}

	return false
}

func (s *Service) recordFailure(result *NamespaceResult, stage string, customerID string, err error) error {
	result.FailureCount++

	if s.logger != nil {
		if customerID == "" {
			s.logger.Warn(
				"ledger account backfill step failed",
				"namespace", result.Namespace,
				"stage", stage,
				"error", err,
			)
		} else {
			s.logger.Warn(
				"ledger account backfill step failed",
				"namespace", result.Namespace,
				"customer_id", customerID,
				"stage", stage,
				"error", err,
			)
		}
	}

	if customerID == "" {
		return fmt.Errorf("namespace=%s stage=%s: %w", result.Namespace, stage, err)
	}

	return fmt.Errorf("namespace=%s customer_id=%s stage=%s: %w", result.Namespace, customerID, stage, err)
}
