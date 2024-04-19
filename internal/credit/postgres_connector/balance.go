package postgres_connector

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/streaming"
	credit_model "github.com/openmeterio/openmeter/pkg/credit"
	"github.com/openmeterio/openmeter/pkg/models"
	product_model "github.com/openmeterio/openmeter/pkg/product"
)

type balanceQueryPeriod struct {
	From time.Time
	To   time.Time
}

func (a *PostgresConnector) GetBalance(
	ctx context.Context,
	namespace string,
	subject string,
	cutline time.Time,
) (credit_model.Balance, error) {
	// Get high watermark for credit
	hw, err := a.GetHighWatermark(ctx, namespace, subject)
	if err != nil {
		return credit_model.Balance{}, fmt.Errorf("get high watermark: %w", err)
	}

	balance, _, err := a.getBalance(ctx, namespace, subject, hw.Time, cutline)
	return balance, err
}

func (a *PostgresConnector) getBalance(
	ctx context.Context,
	namespace string,
	subject string,
	from time.Time,
	to time.Time,
) (credit_model.Balance, credit_model.LedgerEntryList, error) {
	ledgerEntries := credit_model.NewLedgerEntryList()

	grants, err := a.ListGrants(ctx, namespace, credit.ListGrantsParams{
		Subjects:    []string{subject},
		From:        &from,
		To:          &to,
		IncludeVoid: true,
	})
	if err != nil {
		return credit_model.Balance{}, ledgerEntries, fmt.Errorf("list grants: %w", err)
	}

	// Get products in grants
	products := map[string]product_model.Product{}
	for _, grant := range grants {
		if grant.Void {
			ledgerEntries.AddVoidGrant(grant)
			continue
		}

		ledgerEntries.AddGrant(grant)

		if grant.ProductID != nil {
			productID := *grant.ProductID
			if _, ok := products[productID]; !ok {
				product, err := a.GetProduct(ctx, namespace, productID)
				if err != nil {
					return credit_model.Balance{}, ledgerEntries, fmt.Errorf("get product: %w", err)
				}
				products[productID] = product
			}
		}
	}

	// Get meters for products
	// TODO: after we use Ent we can fetch the two together
	meters := map[string]models.Meter{}
	for _, product := range products {
		meterSlug := product.MeterSlug
		if _, ok := meters[meterSlug]; !ok {
			meter, err := a.meterRepository.GetMeterByIDOrSlug(ctx, namespace, meterSlug)
			if err != nil {
				return credit_model.Balance{}, ledgerEntries, fmt.Errorf("get meter: %w", err)
			}
			meters[meterSlug] = meter
		}
	}

	// Calculate usage periods
	// We break down the time range into periods where grants are effective or expire.
	// We need to do this to burn down grants in the correct order.

	// Find pivot dates first (effective and expiration dates in range)
	dates := []time.Time{}
	grantBalances := []credit_model.GrantBalance{}
	for _, grant := range grants {
		if grant.Void {
			continue
		}

		grantBalances = append(grantBalances, credit_model.GrantBalance{
			Grant:   grant,
			Balance: grant.Amount,
		})

		expiresAt := grant.ExpirationDate()

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
	// 2. Grants with product has higher priority
	// 3. Grants with earlier expiration date are burned down first

	// 3. Grants with earlier expiration date are burned down first
	sort.Slice(grantBalances, func(i, j int) bool {
		return grantBalances[i].ExpirationDate().Unix() < grantBalances[j].ExpirationDate().Unix()
	})

	// 2. Order grant balances by product
	// grants with product are applied first
	sort.Slice(grantBalances, func(i, j int) bool {
		a := 1
		b := 1
		if grantBalances[i].ProductID != nil {
			a = 0
		}
		if grantBalances[j].ProductID != nil {
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
			var product *product_model.Product
			grantBalance := &grantBalances[i]

			// Skip grants that does not apply to this period
			expiresAt := grantBalance.ExpirationDate()
			if expiresAt.Before(period.From) {
				continue
			}
			if grantBalance.EffectiveAt.After(period.To) {
				continue
			}

			// Grants without product are not implemented yet
			if grantBalance.ProductID == nil {
				return credit_model.Balance{}, ledgerEntries, fmt.Errorf("not implemented: grants without product")
			}

			// Get product
			if _, ok := products[*grantBalance.ProductID]; !ok {
				return credit_model.Balance{}, ledgerEntries, fmt.Errorf("product not found: %s", *grantBalance.ProductID)
			}
			p := products[*grantBalance.ProductID]
			product = &p

			// Get meter
			if _, ok := meters[product.MeterSlug]; !ok {
				return credit_model.Balance{}, ledgerEntries, fmt.Errorf("meter not found: %s", product.MeterSlug)
			}
			meter := meters[product.MeterSlug]

			// Usage query params for the meter
			queryParams := &streaming.QueryParams{
				FilterSubject: []string{subject},
			}
			queryParams.From = &period.From
			queryParams.To = &period.To
			queryParams.Aggregation = meter.Aggregation

			if product.MeterGroupByFilters != nil {
				queryParams.FilterGroupBy = map[string][]string{}
				for k, v := range *product.MeterGroupByFilters {
					queryParams.FilterGroupBy[k] = []string{v}
				}
			}

			var amount float64 = 0

			// Query cache helps to minimize the number of usage queries between different products with the same meter and group by filters
			queryCacheKey := queryKeyByParams(meter.Slug, queryParams.FilterGroupBy)
			if _, ok := queryCache[queryCacheKey]; !ok {
				// Query usage
				rows, err := a.streamingConnector.QueryMeter(ctx, namespace, meter.Slug, queryParams)
				if err != nil {
					return credit_model.Balance{}, ledgerEntries, fmt.Errorf("query meter: %w", err)
				}
				if len(rows) > 1 {
					return credit_model.Balance{}, ledgerEntries, fmt.Errorf("unexpected number of usage rows")
				}

				// Get usage amount
				if len(rows) == 1 {
					queryCache[queryCacheKey] = rows[0].Value
				} else {
					queryCache[queryCacheKey] = 0
				}
			}

			// Add carry over from previous grant if any, otherwise use the full amount from the query
			carryOverKey := fmt.Sprintf("%s-%s", queryCacheKey, *product.ID)
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

	// Aggregate grant balances by product
	productBalancesMap := map[string]credit_model.ProductBalance{}
	for _, grantBalance := range grantBalances {
		if grantBalance.ProductID == nil {
			continue
		}
		productId := *grantBalance.ProductID
		product := products[productId]

		if productBalance, ok := productBalancesMap[productId]; ok {
			productBalance.Balance += grantBalance.Balance
			productBalancesMap[productId] = productBalance
		} else {
			productBalancesMap[productId] = credit_model.ProductBalance{
				Product: product,
				Balance: grantBalance.Balance,
			}
		}
	}

	// Convert map to slice
	productBalances := []credit_model.ProductBalance{}
	for _, productBalance := range productBalancesMap {
		productBalances = append(productBalances, productBalance)
	}

	return credit_model.Balance{
		Subject:         subject,
		ProductBalances: productBalances,
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
