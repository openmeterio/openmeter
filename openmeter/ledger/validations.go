package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateInvariance validates that Debit - Credit = 0 for each currency in the given entries.
func ValidateInvariance(ctx context.Context, entries []EntryInput) error {
	totals := make(map[string]alpacadecimal.Decimal)
	currencyByKey := make(map[string]currencies.CurrencyReference)
	currencyKeys := make([]string, 0)

	for _, entry := range entries {
		currency := entry.PostingAddress().Route().Route().Currency
		currencyKey := currency.IdentityKey()

		total, ok := totals[currencyKey]
		if !ok {
			currencyKeys = append(currencyKeys, currencyKey)
			currencyByKey[currencyKey] = currency
			total = alpacadecimal.NewFromInt(0)
		}

		totals[currencyKey] = total.Add(entry.Amount())
	}

	for _, currencyKey := range currencyKeys {
		total := totals[currencyKey]
		if total.IsZero() {
			continue
		}

		return ErrInvalidTransactionTotal.WithAttrs(models.Attributes{
			"currency": currencyByKey[currencyKey],
			"total":    total,
			"entries":  entries,
		})
	}

	return nil
}

func ValidateRouting(ctx context.Context, entries []EntryInput) error {
	// Routing validation is implementation-specific and can be injected by the concrete ledger.
	return nil
}

func ValidateEntryInput(ctx context.Context, entry EntryInput) error {
	if entry == nil {
		return ErrEntryInvalid.WithAttrs(models.Attributes{
			"reason": "entry_required",
		})
	}

	// Let's validate the address
	if err := ValidateAddress(ctx, entry.PostingAddress()); err != nil {
		return ErrEntryInvalid.WithAttrs(models.Attributes{
			"reason": "invalid_address",
			"error":  err,
		})
	}

	if err := validateEntryAmountPrecision(entry); err != nil {
		return err
	}

	if err := ValidateEntryIdentityKey(entry); err != nil {
		return ErrEntryInvalid.WithAttrs(models.Attributes{
			"reason": "invalid_identity_key",
			"error":  err,
		})
	}

	return nil
}

// validateEntryAmountPrecision keeps postings at their currency's declared
// minor-unit precision. Fiat precision comes from the ISO currency table.
// Custom currency precision comes from the route's persisted reference
// snapshot, never from a currency-service lookup.
func validateEntryAmountPrecision(entry EntryInput) error {
	route := entry.PostingAddress().Route().Route()
	currencyReference := route.Currency
	currencyCode := currencyReference.Code

	var currency currencyx.Currency
	if currencyCode.IsCustom() {
		resolved, ok := currencyReference.CustomCurrency()
		if !ok {
			return ErrCurrencyInvalid.WithAttrs(models.Attributes{
				"currency": currencyCode,
				"reason":   "custom_currency_reference_must_be_resolved",
			})
		}
		currency = resolved.Currency
	} else {
		built, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
			WithCode(currencyCode).
			Build()
		if err != nil {
			return ErrCurrencyInvalid.WithAttrs(models.Attributes{
				"currency": currencyCode,
				"error":    err,
			})
		}
		currency = built
	}

	amount := entry.Amount()
	if currency.IsRoundedToPrecision(amount) {
		return nil
	}

	return ErrTransactionAmountInvalid.WithAttrs(models.Attributes{
		"reason":         "amount_not_rounded_to_currency_precision",
		"currency":       currency.Details().Code,
		"amount":         amount.String(),
		"rounded_amount": currency.RoundToPrecision(amount).String(),
	})
}

func ValidateAddress(ctx context.Context, address PostingAddress) error {
	if address == nil {
		return ErrAddressInvalid.WithAttrs(models.Attributes{
			"reason": "address_required",
		})
	}

	return nil
}

func ValidateTransactionInput(ctx context.Context, transaction TransactionInput) error {
	return ValidateTransactionInputWith(ctx, transaction, nil)
}

func ValidateTransactionInputWith(ctx context.Context, transaction TransactionInput, routingValidator RoutingValidator) error {
	if transaction == nil {
		return ErrTransactionInputRequired
	}

	entries := lo.Map(transaction.EntryInputs(), func(e EntryInput, _ int) EntryInput {
		return e
	})

	for _, entry := range entries {
		if err := ValidateEntryInput(ctx, entry); err != nil {
			return err
		}
	}

	if err := ValidateInvariance(ctx, entries); err != nil {
		return err
	}

	if err := ValidateOriginProvenance(entries); err != nil {
		return err
	}

	if routingValidator != nil {
		if err := routingValidator.ValidateEntries(transaction.EntryInputs()); err != nil {
			return err
		}
	}

	return nil
}

// ValidateOriginProvenance prevents a balanced transaction from silently moving
// value between collection occurrences or dropping its immutable spend identity.
// Cost-basis translation may add/remove source attribution on the same account
// type; a transfer between account types must preserve the source as well.
func ValidateOriginProvenance(entries []EntryInput) error {
	type originBalance struct {
		amount        alpacadecimal.Decimal
		spend         string
		accountType   AccountType
		sourceAmounts map[string]alpacadecimal.Decimal
		mixedAccount  bool
	}

	balances := make(map[string]originBalance)

	var errs []error

	for _, entry := range entries {
		provenance := entry.Provenance()
		if provenance.CollectionOriginID == nil {
			continue
		}

		spend := lo.FromPtr(provenance.SpendChargeID)
		if spend == "" {
			errs = append(errs, errors.New("origin provenance requires spend_charge_id"))
		}

		key := *provenance.CollectionOriginID + ":" + entry.PostingAddress().Route().Route().Currency.IdentityKey()

		value, exists := balances[key]
		if !exists {
			value.spend = spend
			value.sourceAmounts = make(map[string]alpacadecimal.Decimal)
			value.accountType = entry.PostingAddress().AccountType()
		}

		if value.spend != spend {
			errs = append(errs, errors.New("origin provenance must preserve spend_charge_id"))
		}

		source := lo.FromPtr(provenance.SourceChargeID)
		value.sourceAmounts[source] = value.sourceAmounts[source].Add(entry.Amount())
		value.mixedAccount = value.mixedAccount || value.accountType != entry.PostingAddress().AccountType()
		value.amount = value.amount.Add(entry.Amount())
		balances[key] = value
	}

	for _, value := range balances {
		if !value.amount.IsZero() {
			errs = append(errs, fmt.Errorf("origin provenance must balance independently: %s", value.amount))
		}

		for _, amount := range value.sourceAmounts {
			if amount.IsZero() {
				continue
			}

			if value.mixedAccount {
				errs = append(errs, errors.New("origin transfer must preserve source_charge_id"))
				break
			}

			// The only source-changing operation is attribution of previously unknown
			// receivable/accrued value (or its exact reversal), never purchase A to B.
			_, hasUnknownSource := value.sourceAmounts[""]
			if !hasUnknownSource || len(value.sourceAmounts) != 2 ||
				(value.accountType != AccountTypeCustomerReceivable && value.accountType != AccountTypeCustomerAccrued) {
				errs = append(errs, errors.New("origin source translation must attribute unknown receivable or accrued value"))
				break
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
