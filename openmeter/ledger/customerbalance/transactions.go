package customerbalance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/currencies"
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
	CreditTransactionTypeVoided   CreditTransactionType = "voided"
)

func (t CreditTransactionType) Validate() error {
	switch t {
	case CreditTransactionTypeFunded, CreditTransactionTypeConsumed, CreditTransactionTypeExpired, CreditTransactionTypeVoided:
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

	FeatureFilter mo.Option[creditpurchase.FeatureFilters]
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

	if err := ValidateFeatureFilter(i.FeatureFilter); err != nil {
		errs = append(errs, fmt.Errorf("feature filter: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreditTransaction struct {
	ID          models.NamespacedID
	CreatedAt   time.Time
	BookedAt    time.Time
	Type        CreditTransactionType
	GrantVoided bool
	Currency    currencyx.Code
	Amount      alpacadecimal.Decimal
	Balance     CreditTransactionBalance
	Name        string
	Description *string
	Annotations models.Annotations

	balanceCursor     *ledger.TransactionCursor
	balanceAsOf       *time.Time
	balanceImpact     *alpacadecimal.Decimal
	currencyReference currencies.CurrencyReference

	fundedTransactionGroupID string
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

	accountIDs, err := s.customerBalanceAccountIDs(ctx, input.CustomerID)
	if err != nil {
		return ListCreditTransactionsResult{}, fmt.Errorf("resolve customer balance accounts: %w", err)
	}
	if accountIDs.FBO == "" {
		return emptyCreditTransactions(), nil
	}

	loaders, err := s.creditTransactionLoaders(input.Type)
	if err != nil {
		return ListCreditTransactionsResult{}, err
	}

	loaderInput := creditTransactionLoaderInput{
		Limit:               input.Limit,
		After:               input.After,
		Before:              input.Before,
		CustomerID:          input.CustomerID,
		AccountID:           accountIDs.FBO,
		ReceivableAccountID: accountIDs.Receivable,
		Currency:            input.Currency,
		AsOf:                creditTransactionsAsOf(input.AsOf),
		FeatureFilter:       normalizeFeatureFilter(input.FeatureFilter),
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

	// CurrencyReference.IdentityKey keeps custom currency IDs distinct without
	// using pointer-bearing references as map keys.
	runningBalancesByCurrency := make(map[string]alpacadecimal.Decimal)
	for _, item := range items {
		currencyReference := item.balanceCurrencyReference()
		currencyKey := currencyReference.IdentityKey()
		if _, ok := runningBalancesByCurrency[currencyKey]; ok {
			continue
		}

		runningBalance, err := s.GetSettledBalance(ctx, GetBalanceServiceInput{
			CustomerID:        input.CustomerID,
			Currency:          currencyReference.GetCode(),
			FeatureFilter:     normalizeFeatureFilter(input.FeatureFilter),
			BalanceQuery:      item.balanceQuery(),
			currencyReference: currencyReference,
		})
		if err != nil {
			return ListCreditTransactionsResult{}, fmt.Errorf("get FBO balance after transaction %s: %w", item.ID.ID, err)
		}

		runningBalancesByCurrency[currencyKey] = runningBalance
	}
	applyCreditTransactionBalances(items, runningBalancesByCurrency)

	var (
		nextCursor     *ledger.TransactionCursor
		previousCursor *ledger.TransactionCursor
	)
	if len(items) > 0 {
		lastCursor := creditTransactionCursor(items[len(items)-1])
		firstCursor := creditTransactionCursor(items[0])

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

type customerBalanceAccounts struct {
	FBO        string
	Receivable string
}

func (s *service) customerBalanceAccountIDs(ctx context.Context, customerID customer.CustomerID) (customerBalanceAccounts, error) {
	accounts, err := s.AccountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return customerBalanceAccounts{}, err
	}

	if accounts.FBOAccount == nil {
		return customerBalanceAccounts{}, nil
	}

	accountIDs := customerBalanceAccounts{FBO: accounts.FBOAccount.ID().ID}
	if accounts.ReceivableAccount != nil {
		accountIDs.Receivable = accounts.ReceivableAccount.ID().ID
	}

	return accountIDs, nil
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
	fboImpact, currencyReference, err := creditTransactionFBOImpact(tx)
	if err != nil {
		return CreditTransaction{}, err
	}
	cursor := tx.Cursor()

	return CreditTransaction{
		ID:                tx.ID(),
		CreatedAt:         tx.Cursor().CreatedAt,
		BookedAt:          tx.BookedAt(),
		Type:              creditTransactionType(fboImpact),
		Currency:          currencyReference.GetCode(),
		Amount:            fboImpact,
		Name:              "",
		Annotations:       tx.Annotations(),
		balanceCursor:     &cursor,
		currencyReference: currencyReference,
	}, nil
}

func creditTransactionFBOImpact(tx ledger.Transaction) (alpacadecimal.Decimal, currencies.CurrencyReference, error) {
	amount := ledger.TransactionImpact(tx, ledger.ImpactFilter{
		AccountType: ledger.AccountTypeCustomerFBO,
	})
	var currency currencies.CurrencyReference

	for _, entry := range tx.Entries() {
		if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO {
			continue
		}

		entryCurrency := entry.PostingAddress().Route().Route().Currency
		if currency.Code == "" {
			currency = entryCurrency
		}
		if !currency.Equal(entryCurrency) {
			return alpacadecimal.Decimal{}, currencies.CurrencyReference{}, fmt.Errorf("transaction %s has multiple customer FBO currencies", tx.ID().ID)
		}
	}

	if currency.Code == "" {
		return alpacadecimal.Decimal{}, currencies.CurrencyReference{}, fmt.Errorf("no customer FBO entry found in transaction %s", tx.ID().ID)
	}

	return amount, currency, nil
}

func applyCreditTransactionBalances(items []CreditTransaction, runningBalancesByCurrency map[string]alpacadecimal.Decimal) {
	for i := range items {
		currencyKey := items[i].balanceCurrencyReference().IdentityKey()
		runningBalance := runningBalancesByCurrency[currencyKey]
		items[i].Balance.After = runningBalance
		impact := lo.FromPtrOr(items[i].balanceImpact, items[i].Amount)
		items[i].Balance.Before = runningBalance.Sub(impact)
		runningBalancesByCurrency[currencyKey] = runningBalance.Sub(impact)
	}
}

func (tx CreditTransaction) balanceCurrencyReference() currencies.CurrencyReference {
	if tx.currencyReference.Code != "" {
		return tx.currencyReference.Clone()
	}

	return currencies.NewCurrencyReference(tx.Currency)
}

func (tx CreditTransaction) balanceQuery() ledger.BalanceQuery {
	if tx.balanceCursor != nil {
		return ledger.BalanceQuery{After: tx.balanceCursor}
	}

	if tx.balanceAsOf != nil {
		return ledger.BalanceQuery{AsOf: tx.balanceAsOf}
	}

	asOf := tx.BookedAt
	return ledger.BalanceQuery{AsOf: &asOf}
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

		intent := flatFeeCharge.Intent.GetEffectiveMetaIntentMutableFields()

		return chargeDisplayMetadata{
			Name:        intent.Name,
			Description: intent.Description,
		}, nil
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		if err != nil {
			return chargeDisplayMetadata{}, fmt.Errorf("map usage based charge: %w", err)
		}

		intent := usageBasedCharge.Intent.GetEffectiveMetaIntentMutableFields()

		return chargeDisplayMetadata{
			Name:        intent.Name,
			Description: intent.Description,
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
