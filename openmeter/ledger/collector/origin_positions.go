package collector

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

const unknownOriginSource = "uncovered"

// readOriginPositions reads amounts from the ledger. Original postings supply
// order and reversal routes only; their template codes do not define the state.
// Include all committed movements, as correction may itself be backdated.
func (c *accrualCorrector) readOriginPositions(ctx context.Context, input CorrectCollectedAccruedInput, id string, history originReferences, original *originPair) ([]correctionPosition, error) {
	buckets, err := c.deps.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: input.Namespace,
		Filters:   ledger.Filters{CollectionOriginID: mo.Some(&id)},
		GroupBy:   []string{ledger.BalanceBucketGroupBySourceChargeID, ledger.BalanceBucketGroupBySpendChargeID},
	})
	if err != nil {
		return nil, err
	}
	positions := make(map[string]correctionPosition)
	conservation := make(map[string]alpacadecimal.Decimal)
	for _, bucket := range buckets {
		if lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID]) != input.ChargeID {
			return nil, fmt.Errorf("collection origin belongs to a different spend charge")
		}
		currency := bucket.Address.Route().Route().Currency.IdentityKey()
		conservation[currency] = conservation[currency].Add(bucket.SettledAmount)
		key := lo.FromPtrOr(bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID], unknownOriginSource)
		p := positions[key]
		p.id = key
		p.uncovered = key == unknownOriginSource
		switch bucket.Address.AccountType() {
		case ledger.AccountTypeCustomerAccrued:
			p.accrued = p.accrued.Add(bucket.SettledAmount)
		case ledger.AccountTypeEarnings:
			p.earnings = p.earnings.Add(bucket.SettledAmount)
		case ledger.AccountTypeCustomerReceivable:
			if original.role == originRoleCoverage {
				p.coverage = p.coverage.Add(bucket.SettledAmount)
			}
		}
		positions[key] = p
	}
	for currency, amount := range conservation {
		if !amount.IsZero() {
			return nil, fmt.Errorf("collection origin does not balance for currency %s: %s", currency, amount)
		}
	}
	// Use the first backing occurrence even after later recognition or partial
	// correction. History is newest first, so older evidence replaces newer.
	for _, pair := range history.pairs {
		if pair.role != originRoleBacking && pair != original {
			continue
		}
		key := lo.FromPtrOr(pair.credit.SourceChargeID(), unknownOriginSource)
		p := positions[key]
		p.recordedAt = pair.transaction.Cursor().CreatedAt
		p.orderKey = pair.transaction.ID().ID
		positions[key] = p
	}
	var out []correctionPosition
	for _, p := range positions {
		if !p.amount().IsZero() {
			out = append(out, p)
		}
	}
	return out, nil
}
