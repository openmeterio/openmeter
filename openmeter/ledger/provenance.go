package ledger

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	AnnotationOriginTracked          = "ledger.origin_tracked"
	AnnotationBackfillCreditPriority = "ledger.backfill.credit_priority"
)

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
		if entry.OriginID() == nil {
			continue
		}
		spend := lo.FromPtr(entry.SpendChargeID())
		if spend == "" {
			errs = append(errs, errors.New("origin provenance requires spend_charge_id"))
		}
		key := *entry.OriginID() + ":" + entry.PostingAddress().Route().Route().Currency.IdentityKey()
		value, exists := balances[key]
		if !exists {
			value.spend = spend
			value.sourceAmounts = make(map[string]alpacadecimal.Decimal)
			value.accountType = entry.PostingAddress().AccountType()
		}
		if value.spend != spend {
			errs = append(errs, errors.New("origin provenance must preserve spend_charge_id"))
		}
		source := lo.FromPtr(entry.SourceChargeID())
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
