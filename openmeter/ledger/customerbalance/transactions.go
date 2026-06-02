package customerbalance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreditTransactionType string

const (
	CreditTransactionTypeFunded   CreditTransactionType = "funded"
	CreditTransactionTypeConsumed CreditTransactionType = "consumed"
	CreditTransactionTypeExpired  CreditTransactionType = "expired"
)

func (t CreditTransactionType) Validate() error {
	switch t {
	case CreditTransactionTypeFunded, CreditTransactionTypeConsumed, CreditTransactionTypeExpired:
		return nil
	default:
		return fmt.Errorf("invalid credit transaction type: %s", t)
	}
}

type ListCreditTransactionsInput struct {
	CustomerID customer.CustomerID
	Limit      int
	After      *ledger.TransactionCursor
	Before     *ledger.TransactionCursor

	Type     *CreditTransactionType
	Currency *currencyx.Code
	AsOf     *time.Time
}

func (i ListCreditTransactionsInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.Limit < 1 {
		errs = append(errs, fmt.Errorf("limit must be greater than 0"))
	}

	if i.After != nil {
		if err := i.After.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("after: %w", err))
		}
	}

	if i.Before != nil {
		if err := i.Before.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("before: %w", err))
		}
	}

	if i.After != nil && i.Before != nil {
		errs = append(errs, fmt.Errorf("after and before cannot be set together"))
	}

	if i.Type != nil {
		if err := i.Type.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("type: %w", err))
		}
	}

	if i.Currency != nil {
		if err := i.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}

	if i.AsOf != nil && i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("asOf must not be zero"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreditTransaction struct {
	ID          models.NamespacedID
	CreatedAt   time.Time
	BookedAt    time.Time
	Type        CreditTransactionType
	Currency    currencyx.Code
	Amount      alpacadecimal.Decimal
	Balance     CreditTransactionBalance
	Name        string
	Description *string
	Annotations models.Annotations
}

type CreditTransactionBalance struct {
	Before alpacadecimal.Decimal
	After  alpacadecimal.Decimal
}

type ListCreditTransactionsResult struct {
	Items          []CreditTransaction
	NextCursor     *ledger.TransactionCursor
	PreviousCursor *ledger.TransactionCursor
}

func (s *service) ListCreditTransactions(ctx context.Context, input ListCreditTransactionsInput) (ListCreditTransactionsResult, error) {
	if err := input.Validate(); err != nil {
		return ListCreditTransactionsResult{}, err
	}

	accountID, err := s.customerFBOAccountID(ctx, input.CustomerID)
	if err != nil {
		return ListCreditTransactionsResult{}, fmt.Errorf("resolve customer FBO account: %w", err)
	}
	if accountID == "" {
		return emptyCreditTransactions(), nil
	}

	loaders, err := s.creditTransactionLoaders(input.Type)
	if err != nil {
		return ListCreditTransactionsResult{}, err
	}

	loaderInput := creditTransactionLoaderInput{
		Limit:      input.Limit,
		After:      input.After,
		Before:     input.Before,
		CustomerID: input.CustomerID,
		AccountID:  accountID,
		Currency:   input.Currency,
		AsOf:       creditTransactionsAsOf(input.AsOf),
	}

	loadedLists := make([][]CreditTransaction, 0, len(loaders))
	loadersHaveMore := false
	for i, loader := range loaders {
		loaded, err := loader.Load(ctx, loaderInput)
		if err != nil {
			return ListCreditTransactionsResult{}, fmt.Errorf("load transactions from loader %d: %w", i, err)
		}

		loadedLists = append(loadedLists, loaded.Items)
		loadersHaveMore = loadersHaveMore || loaded.HasMore
	}

	mergedItems, bufferedHasMore := mergeSortedLists(
		loadedLists,
		input.Limit,
		compareCreditTransactionsByCursor,
	)
	// bufferedHasMore only reflects whether there are still items in the fetched in-memory lists.
	// loadersHaveMore captures additional records in the requested cursor direction beyond each loader's in-memory window.
	hasMoreInQueryDirection := bufferedHasMore || loadersHaveMore

	items := mergedItems
	s.applyChargeMetadataToCreditTransactions(ctx, input.CustomerID.Namespace, items)

	if len(items) > 0 {
		runningBalance, err := s.GetBalance(ctx, input.CustomerID, items[0].Currency, ledger.BalanceQuery{
			After: lo.ToPtr(creditTransactionCursor(items[0])),
		})
		if err != nil {
			return ListCreditTransactionsResult{}, fmt.Errorf("get FBO balance after transaction %s: %w", items[0].ID.ID, err)
		}

		applyCreditTransactionBalances(items, runningBalance.Settled())
	}

	var (
		nextCursor     *ledger.TransactionCursor
		previousCursor *ledger.TransactionCursor
	)
	if len(mergedItems) > 0 {
		lastCursor := creditTransactionCursor(mergedItems[len(mergedItems)-1])
		firstCursor := creditTransactionCursor(mergedItems[0])

		if input.Before != nil || hasMoreInQueryDirection {
			nextCursor = lo.ToPtr(lastCursor)
		}

		if (input.Before != nil && hasMoreInQueryDirection) || input.After != nil {
			previousCursor = lo.ToPtr(firstCursor)
		}
	}

	return ListCreditTransactionsResult{
		Items:          items,
		NextCursor:     nextCursor,
		PreviousCursor: previousCursor,
	}, nil
}

func emptyCreditTransactions() ListCreditTransactionsResult {
	return ListCreditTransactionsResult{
		Items: []CreditTransaction{},
	}
}

func creditTransactionsAsOf(asOf *time.Time) time.Time {
	if asOf != nil {
		return *asOf
	}

	return clock.Now()
}

func (s *service) customerFBOAccountID(ctx context.Context, customerID customer.CustomerID) (string, error) {
	accounts, err := s.AccountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return "", err
	}

	if accounts.FBOAccount == nil {
		return "", nil
	}

	return accounts.FBOAccount.ID().ID, nil
}

func creditTransactionsFromLedgerTransactions(txs []ledger.Transaction) ([]CreditTransaction, error) {
	items := make([]CreditTransaction, 0, len(txs))

	for _, tx := range txs {
		item, err := creditTransactionFromLedgerTransaction(tx)
		if err != nil {
			return nil, fmt.Errorf("convert ledger transaction %s: %w", tx.ID().ID, err)
		}

		items = append(items, item)
	}

	return items, nil
}

func creditTransactionFromLedgerTransaction(tx ledger.Transaction) (CreditTransaction, error) {
	fboImpact, currency, err := creditTransactionFBOImpact(tx)
	if err != nil {
		return CreditTransaction{}, err
	}

	return CreditTransaction{
		ID:          tx.ID(),
		CreatedAt:   tx.Cursor().CreatedAt,
		BookedAt:    tx.BookedAt(),
		Type:        creditTransactionType(fboImpact),
		Currency:    currency,
		Amount:      fboImpact,
		Name:        "",
		Annotations: tx.Annotations(),
	}, nil
}

func creditTransactionFBOImpact(tx ledger.Transaction) (alpacadecimal.Decimal, currencyx.Code, error) {
	amount := ledger.TransactionImpact(tx, ledger.ImpactFilter{
		AccountType: ledger.AccountTypeCustomerFBO,
	})
	var currency currencyx.Code

	for _, entry := range tx.Entries() {
		if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO {
			continue
		}

		entryCurrency := entry.PostingAddress().Route().Route().Currency
		if currency == "" {
			currency = entryCurrency
		}
		if currency != entryCurrency {
			return alpacadecimal.Decimal{}, "", fmt.Errorf("transaction %s has multiple customer FBO currencies", tx.ID().ID)
		}
	}

	if currency == "" {
		return alpacadecimal.Decimal{}, "", fmt.Errorf("no customer FBO entry found in transaction %s", tx.ID().ID)
	}

	return amount, currency, nil
}

func applyCreditTransactionBalances(items []CreditTransaction, after alpacadecimal.Decimal) {
	runningBalance := after

	for i := range items {
		items[i].Balance.After = runningBalance
		items[i].Balance.Before = runningBalance.Sub(items[i].Amount)
		runningBalance = runningBalance.Sub(items[i].Amount)
	}
}

func creditTransactionType(fboImpact alpacadecimal.Decimal) CreditTransactionType {
	if fboImpact.IsPositive() {
		return CreditTransactionTypeFunded
	}

	return CreditTransactionTypeConsumed
}

type chargeDisplayMetadata struct {
	Name        string
	Description *string
}

func (s *service) applyChargeMetadataToCreditTransactions(ctx context.Context, namespace string, items []CreditTransaction) {
	chargeIDs := lo.Uniq(lo.FilterMap(items, func(item CreditTransaction, _ int) (string, bool) {
		id := chargeIDFromAnnotations(item.Annotations)
		return id, id != ""
	}))

	if len(chargeIDs) == 0 {
		return
	}

	chargeEntities, err := s.ChargesService.GetByIDs(ctx, charges.GetByIDsInput{
		Namespace: namespace,
		IDs:       chargeIDs,
	})
	if err != nil {
		return
	}

	chargeDisplayByID := make(map[string]chargeDisplayMetadata, len(chargeEntities))
	for _, chargeEntity := range chargeEntities {
		chargeID, err := chargeEntity.GetChargeID()
		if err != nil {
			continue
		}

		metadata, err := chargeDisplayMetadataFromCharge(chargeEntity)
		if err != nil {
			continue
		}

		chargeDisplayByID[chargeID.ID] = metadata
	}

	for i := range items {
		chargeID := chargeIDFromAnnotations(items[i].Annotations)
		if chargeID == "" {
			continue
		}

		metadata, ok := chargeDisplayByID[chargeID]
		if !ok {
			continue
		}

		items[i].Name = metadata.Name
		items[i].Description = metadata.Description
	}
}

func chargeDisplayMetadataFromCharge(charge charges.Charge) (chargeDisplayMetadata, error) {
	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		if err != nil {
			return chargeDisplayMetadata{}, fmt.Errorf("map flat fee charge: %w", err)
		}

		return chargeDisplayMetadata{
			Name:        flatFeeCharge.Intent.Name,
			Description: flatFeeCharge.Intent.Description,
		}, nil
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		if err != nil {
			return chargeDisplayMetadata{}, fmt.Errorf("map usage based charge: %w", err)
		}

		return chargeDisplayMetadata{
			Name:        usageBasedCharge.Intent.Name,
			Description: usageBasedCharge.Intent.Description,
		}, nil
	case meta.ChargeTypeCreditPurchase:
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		if err != nil {
			return chargeDisplayMetadata{}, fmt.Errorf("map credit purchase charge: %w", err)
		}

		return chargeDisplayMetadata{
			Name:        creditPurchaseCharge.Intent.Name,
			Description: creditPurchaseCharge.Intent.Description,
		}, nil
	default:
		return chargeDisplayMetadata{}, fmt.Errorf("unsupported charge type %s", charge.Type())
	}
}

func chargeIDFromAnnotations(annotations models.Annotations) string {
	raw, ok := annotations[ledger.AnnotationChargeID]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return value
}
