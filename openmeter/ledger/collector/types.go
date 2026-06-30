package collector

import (
	"cmp"
	"strconv"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FBO Sources for Prioritization

type fboCollectionSource struct {
	address           ledger.PostingAddress
	sourceChargeID    *string
	available         alpacadecimal.Decimal
	creditPriority    int
	featureRestricted bool
	expiresAt         *time.Time
	cursor            string
	breakagePlan      *breakage.Plan
}

var _ cmpx.Comparable[fboCollectionSource] = fboCollectionSource{}

// TODO: Version this contract before changing it. Corrections and breakage
// releases depend on collection selecting sources deterministically.
func (s fboCollectionSource) Compare(other fboCollectionSource) int {
	if c := cmp.Compare(s.creditPriority, other.creditPriority); c != 0 {
		return c
	}

	if s.featureRestricted != other.featureRestricted {
		if s.featureRestricted {
			return -1
		}

		return 1
	}

	if c := func(left, right *time.Time) int {
		switch {
		case left == nil && right == nil:
			return 0
		case left == nil:
			return 1
		case right == nil:
			return -1
		default:
			return left.Compare(*right)
		}
	}(s.expiresAt, other.expiresAt); c != 0 {
		return c
	}

	return cmp.Compare(s.cursor, other.cursor)
}

// Selections for Consumption Plan

type fboCollectionSelection struct {
	source fboCollectionSource
	amount alpacadecimal.Decimal
}

type fboCollectionSelections []fboCollectionSelection

func (s fboCollectionSelections) postingAmounts(spendChargeID *string) []transactions.PostingAmount {
	out := make([]transactions.PostingAmount, 0, len(s))

	for idx, selection := range s {
		collectionSource := strconv.Itoa(idx)
		out = append(out, transactions.PostingAmount{
			Address: selection.source.address,
			Amount:  selection.amount,
			Identity: ledger.EntryIdentityParts{
				CollectionSource: &collectionSource,
				SourceChargeID:   selection.source.sourceChargeID,
				SpendChargeID:    spendChargeID,
			},
			Annotations: models.Annotations{
				ledger.AnnotationCollectionSourceOrder: idx,
			},
		})
	}

	return out
}
