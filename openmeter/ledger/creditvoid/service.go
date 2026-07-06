package creditvoid

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	VoidCreditPurchase(ctx context.Context, input VoidCreditPurchaseInput) (VoidCreditPurchaseResult, error)
}

type Config struct {
	Ledger             ledger.Ledger
	Dependencies       transactions.ResolverDependencies
	Breakage           breakage.Service
	AccountLocker      ledger.AccountLocker
	TransactionManager transaction.Creator
}

func (c Config) Validate() error {
	var errs []error

	if c.Ledger == nil {
		errs = append(errs, errors.New("ledger is required"))
	}
	if c.Dependencies.AccountService == nil {
		errs = append(errs, errors.New("account service is required"))
	}
	if c.Dependencies.AccountCatalog == nil {
		errs = append(errs, errors.New("account catalog is required"))
	}
	if c.Dependencies.BalanceQuerier == nil {
		errs = append(errs, errors.New("balance querier is required"))
	}
	if c.Breakage == nil {
		errs = append(errs, errors.New("breakage service is required"))
	}
	if c.AccountLocker == nil {
		errs = append(errs, errors.New("account locker is required"))
	}
	if c.TransactionManager == nil {
		errs = append(errs, errors.New("transaction manager is required"))
	}

	return errors.Join(errs...)
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		ledger:             config.Ledger,
		deps:               config.Dependencies,
		breakage:           config.Breakage,
		accountLocker:      config.AccountLocker,
		transactionManager: config.TransactionManager,
	}, nil
}

type service struct {
	ledger             ledger.Ledger
	deps               transactions.ResolverDependencies
	breakage           breakage.Service
	accountLocker      ledger.AccountLocker
	transactionManager transaction.Creator
}

type VoidCreditPurchaseInput struct {
	CustomerID  customer.CustomerID
	ChargeID    string
	Currency    currencyx.Code
	ExpiresAt   *time.Time
	Annotations models.Annotations
}

func (i VoidCreditPurchaseInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	return errors.Join(errs...)
}

type VoidCreditPurchaseResult struct {
	VoidedAt           time.Time
	Amount             alpacadecimal.Decimal
	TransactionGroupID string
}

// voidPlan is the read-only outcome of the planning step: the concrete
// remaining balance slices to forfeit, each with the open expiry breakage plan
// it must release.
type voidPlan struct {
	voidedAt time.Time
	slices   []voidSlice
}

type voidSlice struct {
	amount     alpacadecimal.Decimal
	fboAddress ledger.PostingAddress
	// expiryPlan is nil for non-expiring charges.
	expiryPlan *breakage.Plan
}

func (s *service) VoidCreditPurchase(ctx context.Context, input VoidCreditPurchaseInput) (VoidCreditPurchaseResult, error) {
	if err := input.Validate(); err != nil {
		return VoidCreditPurchaseResult{}, err
	}

	return transaction.Run(ctx, s.transactionManager, func(ctx context.Context) (VoidCreditPurchaseResult, error) {
		plan, err := s.planVoid(ctx, input)
		if err != nil {
			return VoidCreditPurchaseResult{}, err
		}

		return s.executeVoid(ctx, input, plan)
	})
}

// planVoid locks the customer FBO account, reads the charge's remaining
// balance slices, and matches each to the open expiry breakage plan it must
// release. The lock stays held until the surrounding transaction commits, so
// the plan cannot go stale against concurrent collection or voiding.
func (s *service) planVoid(ctx context.Context, input VoidCreditPurchaseInput) (voidPlan, error) {
	// Truncate to Postgres timestamp precision so the returned void time
	// matches the persisted rows byte-for-byte.
	voidedAt := clock.Now().UTC().Truncate(time.Microsecond)

	customerAccounts, err := s.deps.AccountService.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return voidPlan{}, fmt.Errorf("get customer accounts: %w", err)
	}

	if err := s.accountLocker.LockAccountsForPosting(ctx, []ledger.Account{customerAccounts.FBOAccount}); err != nil {
		return voidPlan{}, fmt.Errorf("lock customer FBO account: %w", err)
	}

	fboAccountID := customerAccounts.FBOAccount.ID()
	// The as-of read naturally excludes the charge's own future-dated expiry
	// breakage entries.
	buckets, err := s.deps.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: input.CustomerID.Namespace,
		Filters: ledger.Filters{
			AccountID:      &fboAccountID.ID,
			SourceChargeID: mo.Some(&input.ChargeID),
			AsOf:           &voidedAt,
			Route: ledger.RouteFilter{
				Currency: input.Currency,
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	if err != nil {
		return voidPlan{}, fmt.Errorf("get FBO balance buckets: %w", err)
	}

	openPlansBySubAccount, err := s.openExpiryPlansBySubAccount(ctx, input, voidedAt)
	if err != nil {
		return voidPlan{}, err
	}

	plan := voidPlan{voidedAt: voidedAt}
	for _, bucket := range buckets {
		if !bucket.SettledAmount.IsPositive() {
			continue
		}

		slice := voidSlice{
			amount:     bucket.SettledAmount,
			fboAddress: bucket.Address,
		}

		if openPlansBySubAccount != nil {
			// The expiry plan must cover the voided slice in full, otherwise
			// the breakage bookkeeping already disagrees with the live balance
			// and expiry would remove the same value a second time.
			expiryPlan, ok := openPlansBySubAccount[bucket.Address.SubAccountID()]
			if !ok {
				return voidPlan{}, fmt.Errorf("no open expiry breakage plan for charge %s sub-account %s", input.ChargeID, bucket.Address.SubAccountID())
			}
			if expiryPlan.OpenAmount.LessThan(slice.amount) {
				return voidPlan{}, fmt.Errorf("open expiry breakage plan %s amount %s is less than voided amount %s", expiryPlan.ID.ID, expiryPlan.OpenAmount, slice.amount)
			}

			slice.expiryPlan = &expiryPlan
		}

		plan.slices = append(plan.slices, slice)
	}

	if len(plan.slices) == 0 {
		return voidPlan{}, models.NewGenericConflictError(
			fmt.Errorf("credit purchase %s has no remaining value to void", input.ChargeID),
		)
	}

	return plan, nil
}

// openExpiryPlansBySubAccount returns nil for non-expiring charges.
func (s *service) openExpiryPlansBySubAccount(ctx context.Context, input VoidCreditPurchaseInput, voidedAt time.Time) (map[string]breakage.Plan, error) {
	if input.ExpiresAt == nil || !input.ExpiresAt.After(voidedAt) {
		return nil, nil
	}

	openPlans, err := s.breakage.ListPlans(ctx, breakage.ListPlansInput{
		CustomerID: input.CustomerID,
		Currency:   input.Currency,
		AsOf:       voidedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("list open breakage plans: %w", err)
	}

	out := make(map[string]breakage.Plan)
	for _, plan := range openPlans {
		if plan.SourceChargeID == nil || *plan.SourceChargeID != input.ChargeID {
			continue
		}

		out[plan.FBOSubAccountID] = plan
	}

	return out, nil
}

func (s *service) executeVoid(ctx context.Context, input VoidCreditPurchaseInput, plan voidPlan) (VoidCreditPurchaseResult, error) {
	var (
		inputs  []ledger.TransactionInput
		pending []breakage.PendingRecord
	)
	amount := alpacadecimal.Zero

	for _, slice := range plan.slices {
		voidTx, voidRecord, err := s.breakage.PlanVoid(ctx, breakage.PlanVoidInput{
			CustomerID:     input.CustomerID,
			VoidedAt:       plan.voidedAt,
			Amount:         slice.amount,
			SourceChargeID: input.ChargeID,
			FBOAddress:     slice.fboAddress,
		})
		if err != nil {
			return VoidCreditPurchaseResult{}, fmt.Errorf("resolve void breakage plan: %w", err)
		}

		inputs = append(inputs, voidTx)
		pending = append(pending, voidRecord)
		amount = amount.Add(slice.amount)

		if slice.expiryPlan == nil {
			continue
		}

		releaseTx, releaseRecord, err := s.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{
			Plan:           *slice.expiryPlan,
			Amount:         slice.amount,
			SourceKind:     breakage.SourceKindCreditPurchaseVoid,
			SourceChargeID: &input.ChargeID,
		})
		if err != nil {
			return VoidCreditPurchaseResult{}, fmt.Errorf("resolve expiry breakage release: %w", err)
		}

		inputs = append(inputs, releaseTx)
		pending = append(pending, releaseRecord)
	}

	for i, txInput := range inputs {
		if txInput != nil {
			inputs[i] = transactions.WithAnnotations(txInput, input.Annotations)
		}
	}

	transactionGroup, err := s.ledger.CommitGroup(ctx, transactions.GroupInputs(
		input.CustomerID.Namespace,
		input.Annotations,
		inputs...,
	))
	if err != nil {
		return VoidCreditPurchaseResult{}, fmt.Errorf("commit ledger transaction group: %w", err)
	}

	if err := s.breakage.PersistCommittedRecords(ctx, pending, transactionGroup); err != nil {
		return VoidCreditPurchaseResult{}, fmt.Errorf("persist breakage records: %w", err)
	}

	return VoidCreditPurchaseResult{
		VoidedAt:           plan.voidedAt,
		Amount:             amount,
		TransactionGroupID: transactionGroup.ID().ID,
	}, nil
}
