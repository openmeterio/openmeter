package postgres_connector

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

type balanceQueryPeriod struct {
	From time.Time
	To   time.Time
}

func (a *PostgresConnector) GetBalance(
	ctx context.Context,
	namespace string,
	ledgerID ulid.ULID,
	cutline time.Time,
) (credit.Balance, error) {
	// TODO: wrap into transaction
	hw, err := a.GetHighWatermark(ctx, namespace, ledgerID)
	if err != nil {
		return credit.Balance{}, fmt.Errorf("get high watermark: %w", err)
	}

	balance, _, err := a.getBalance(ctx, namespace, ledgerID, hw.Time, cutline)
	return balance, err
}

func (a *PostgresConnector) getBalance(
	ctx context.Context,
	namespace string,
	ledgerID ulid.ULID,
	from time.Time,
	to time.Time,
) (credit.Balance, credit.LedgerEntryList, error) {
	ledgerEntries := credit.NewLedgerEntryList()

	ledger, err := a.getLedger(ctx, namespace, ledgerID)
	if err != nil {
		return credit.Balance{}, ledgerEntries, err
	}

	grants, err := a.ListGrants(ctx, namespace, credit.ListGrantsParams{
		LedgerIDs:   []ulid.ULID{ledgerID},
		From:        &from,
		To:          &to,
		IncludeVoid: true,
	})
	if err != nil {
		return credit.Balance{}, ledgerEntries, fmt.Errorf("list grants: %w", err)
	}

	// Get features in grants
	features := map[ulid.ULID]credit.Feature{}
	for _, grant := range grants {
		if grant.Void {
			ledgerEntries.AddVoidGrant(grant)
			continue
		}

		ledgerEntries.AddGrant(grant)

		if grant.FeatureID != nil {
			featureID := *grant.FeatureID
			if _, ok := features[featureID]; !ok {
				feature, err := a.GetFeature(ctx, namespace, featureID)
				if err != nil {
					return credit.Balance{}, ledgerEntries, fmt.Errorf("get feature: %w", err)
				}
				features[featureID] = feature
			}
		}
	}

	// Get meters for features
	// TODO: after we use Ent we can fetch the two together
	meters := map[string]models.Meter{}
	for _, feature := range features {
		meterSlug := feature.MeterSlug
		if _, ok := meters[meterSlug]; !ok {
			meter, err := a.meterRepository.GetMeterByIDOrSlug(ctx, namespace, meterSlug)
			if err != nil {
				return credit.Balance{}, ledgerEntries, fmt.Errorf("get meter: %w", err)
			}
			meters[meterSlug] = meter
		}
	}

	// Calculate usage periods
	// We break down the time range into periods where grants are effective or expire.
	// We need to do this to burn down grants in the correct order.

	// Find pivot dates first (effective and expiration dates in range)
	dates := []time.Time{}
	grantBalances := []credit.GrantBalance{}
	for _, grant := range grants {
		if grant.Void {
			continue
		}

		grantBalances = append(grantBalances, credit.GrantBalance{
			Grant:   grant,
			Balance: grant.Amount,
		})

		expiresAt := grant.ExpiresAt

		if (grant.EffectiveAt.After(from) || grant.EffectiveAt.Equal(from)) && (grant.EffectiveAt.Before(to)) {
			dates = append(dates, grant.EffectiveAt)
		}
		if (expiresAt.After(from) || expiresAt.Equal(from)) && (expiresAt.Before(to)) {
			dates = append(dates, expiresAt)
		}
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})
	dates = removeDuplicateTimes(dates)

	// Create periods from pivot dates
	periods := []balanceQueryPeriod{}
	if len(dates) > 0 {
		periodFrom := dates[0]
		dates = dates[1:]
		var periodTo time.Time
		for _, date := range dates {
			periodTo = date
			periods = append(periods, balanceQueryPeriod{From: periodFrom, To: periodTo})
			periodFrom = date
		}
		periods = append(periods, balanceQueryPeriod{From: periodFrom, To: to})
	}

	// The correct order to burn down grants is:
	// 1. Grants with higher priority are burned down first
	// 2. Grants with feature has higher priority
	// 3. Grants with earlier expiration date are burned down first

	// 3. Grants with earlier expiration date are burned down first
	sort.Slice(grantBalances, func(i, j int) bool {
		return grantBalances[i].ExpirationDate().Unix() < grantBalances[j].ExpirationDate().Unix()
	})

	// 2. Order grant balances by feature
	// grants with feature are applied first
	sort.Slice(grantBalances, func(i, j int) bool {
		a := 1
		b := 1
		if grantBalances[i].FeatureID != nil {
			a = 0
		}
		if grantBalances[j].FeatureID != nil {
			b = 0
		}

		return a < b
	})

	// 1. Order grant balances by priority
	sort.Slice(grantBalances, func(i, j int) bool {
		return grantBalances[i].Priority < grantBalances[j].Priority
	})

	// Query usage for each period
	for _, period := range periods {
		queryCache := map[string]float64{}
		carryOverAmount := map[string]float64{}

		for i := range grantBalances {
			var feature *credit.Feature
			grantBalance := &grantBalances[i]

			// Skip grants that does not apply to this period
			expiresAt := grantBalance.ExpirationDate()
			if expiresAt.Before(period.From) {
				continue
			}
			if grantBalance.EffectiveAt.After(period.To) {
				continue
			}

			// Grants without feature are not implemented yet
			if grantBalance.FeatureID == nil {
				return credit.Balance{}, ledgerEntries, fmt.Errorf("not implemented: grants without feature")
			}

			// Get feature
			if _, ok := features[*grantBalance.FeatureID]; !ok {
				return credit.Balance{}, ledgerEntries, fmt.Errorf("feature not found: %s", *grantBalance.FeatureID)
			}
			p := features[*grantBalance.FeatureID]
			feature = &p

			// Get meter
			if _, ok := meters[feature.MeterSlug]; !ok {
				return credit.Balance{}, ledgerEntries, fmt.Errorf("meter not found: %s", feature.MeterSlug)
			}
			meter := meters[feature.MeterSlug]

			// Usage query params for the meter
			queryParams := &streaming.QueryParams{
				// TODO: do we want this to be settable in ledger
				FilterSubject: []string{ledger.Subject},
			}
			queryParams.From = &period.From
			queryParams.To = &period.To
			queryParams.Aggregation = meter.Aggregation

			if feature.MeterGroupByFilters != nil {
				queryParams.FilterGroupBy = map[string][]string{}
				for k, v := range *feature.MeterGroupByFilters {
					queryParams.FilterGroupBy[k] = []string{v}
				}
			}

			var amount float64 = 0

			// Query cache helps to minimize the number of usage queries between different features with the same meter and group by filters
			queryCacheKey := queryKeyByParams(meter.Slug, queryParams.FilterGroupBy)
			if _, ok := queryCache[queryCacheKey]; !ok {
				// Query usage
				rows, err := a.streamingConnector.QueryMeter(ctx, namespace, meter.Slug, queryParams)
				if err != nil {
					return credit.Balance{}, ledgerEntries, fmt.Errorf("query meter: %w", err)
				}
				if len(rows) > 1 {
					return credit.Balance{}, ledgerEntries, fmt.Errorf("unexpected number of usage rows")
				}

				// Get usage amount
				if len(rows) == 1 {
					queryCache[queryCacheKey] = rows[0].Value
				} else {
					queryCache[queryCacheKey] = 0
				}
			}

			// Add carry over from previous grant if any, otherwise use the full amount from the query
			carryOverKey := fmt.Sprintf("%s-%s", queryCacheKey, *feature.ID)
			if val, ok := carryOverAmount[carryOverKey]; ok {
				amount += val
			} else {
				carryOverAmount[carryOverKey] = 0
				amount += queryCache[queryCacheKey]
			}

			// Nothing to do if amount is 0
			if amount == 0 {
				continue
			}

			ledgerTime := period.To
			if ledgerTime.After(time.Now()) {
				ledgerTime = time.Now()
			}
			ledgerAmount := -amount

			// Burn down the grant and apply to the balance
			if amount > grantBalance.Balance {
				amount -= grantBalance.Balance
				ledgerAmount = amount * -1
				grantBalance.Balance = 0
			} else {
				grantBalance.Balance -= amount
				amount = 0
			}

			ledgerEntries.AddGrantUsage(*grantBalance, period.From, ledgerTime, ledgerAmount)

			carryOverAmount[carryOverKey] += amount
		}
	}

	// Aggregate grant balances by feature
	featureBalancesMap := map[ulid.ULID]credit.FeatureBalance{}
	for _, grantBalance := range grantBalances {
		if grantBalance.FeatureID == nil {
			continue
		}
		featureId := *grantBalance.FeatureID
		feature := features[featureId]

		if featureBalance, ok := featureBalancesMap[featureId]; ok {
			featureBalance.Balance += grantBalance.Balance
			featureBalancesMap[featureId] = featureBalance
		} else {
			featureBalancesMap[featureId] = credit.FeatureBalance{
				Feature: feature,
				Balance: grantBalance.Balance,
			}
		}
	}

	// Convert map to slice
	featureBalances := []credit.FeatureBalance{}
	for _, featureBalance := range featureBalancesMap {
		featureBalances = append(featureBalances, featureBalance)
	}

	return credit.Balance{
		LedgerID:        ledgerID,
		Subject:         ledger.Subject,
		Metadata:        ledger.Metadata,
		FeatureBalances: featureBalances,
		GrantBalances:   grantBalances,
	}, ledgerEntries, nil
}

// removeDuplicateTimes removes duplicate dates from the slice
func removeDuplicateTimes(times []time.Time) []time.Time {
	allKeys := make(map[int64]bool)
	list := []time.Time{}
	for _, t := range times {
		key := t.Unix()
		if _, value := allKeys[key]; !value {
			allKeys[key] = true
			list = append(list, t)
		}
	}
	return list
}

// queryKeyByParams generates a unique key for each meter query
func queryKeyByParams(meterSlug string, groupByFilter map[string][]string) string {
	var groupByFilters []string

	for k, v := range groupByFilter {
		groupByFilter := fmt.Sprintf("%s=%s", k, strings.Join(v, ","))
		groupByFilters = append(groupByFilters, groupByFilter)
	}
	slices.Sort(groupByFilters)

	return fmt.Sprintf("%s-%s", meterSlug, strings.Join(groupByFilters, "-"))
}
