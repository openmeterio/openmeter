package advance

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

// A purchase can backfill multiple spends in one group. Match the original
// unknown-cost route and spend rather than taking the group's first template.
type LegacyBackfillTransactionInput struct {
	Group        ledger.TransactionGroup
	Original     ledger.Transaction
	AccountType  ledger.AccountType
	TemplateCode string
}

func (i LegacyBackfillTransactionInput) Validate() error {
	var errs []error

	if i.Group == nil {
		errs = append(errs, errors.New("backing group is required"))
	}

	if i.Original == nil {
		errs = append(errs, errors.New("original transaction is required"))
	}

	if i.AccountType == "" {
		errs = append(errs, errors.New("account type is required"))
	}

	if i.TemplateCode == "" {
		errs = append(errs, errors.New("template code is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// FindLegacyBackfillTransaction matches the original unknown-cost route and spend.
func FindLegacyBackfillTransaction(input LegacyBackfillTransactionInput) (ledger.Transaction, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	keys := make(map[legacyPostingKey]struct{})

	for _, entry := range input.Original.Entries() {
		if entry.PostingAddress().AccountType() == input.AccountType {
			keys[legacyEntryKey(entry)] = struct{}{}
		}
	}

	var found ledger.Transaction

	for _, tx := range input.Group.Transactions() {
		code, err := ledger.TransactionTemplateCodeFromAnnotations(tx.Annotations())
		if err != nil {
			return nil, err
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(tx.Annotations())
		if err != nil {
			return nil, err
		}

		if code != input.TemplateCode || direction != ledger.TransactionDirectionForward {
			continue
		}

		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() != input.AccountType {
				continue
			}

			if _, ok := keys[legacyEntryKey(entry)]; !ok {
				continue
			}

			if found != nil {
				return nil, fmt.Errorf("multiple backfill transactions match the original %s source", input.AccountType)
			}

			found = tx
			break
		}
	}

	return found, nil
}

type legacyPostingKey struct{ subAccountID, sourceChargeID, spendChargeID string }

func legacyEntryKey(entry ledger.EntryInput) legacyPostingKey {
	return legacyPostingKey{
		subAccountID:   entry.PostingAddress().SubAccountID(),
		sourceChargeID: lo.FromPtrOr(entry.Provenance().SourceChargeID, ""),
		spendChargeID:  lo.FromPtrOr(entry.Provenance().SpendChargeID, ""),
	}
}
