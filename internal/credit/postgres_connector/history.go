package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *PostgresConnector) GetHistory(
	ctx context.Context,
	ledgerID credit.NamespacedLedgerID,
	from time.Time,
	to time.Time,
	pagination credit.Pagination,
	windowParams *credit.WindowParams,
) (credit.LedgerEntryList, error) {
	ledgerEntries := credit.NewLedgerEntryList()

	query := a.db.CreditEntry.Query().Where(
		db_credit.And(
			db_credit.EntryTypeEQ(credit.EntryTypeReset),
			db_credit.EffectiveAtGTE(from),
			db_credit.EffectiveAtLTE(to),
		),
	).Order(
		db_credit.ByCreatedAt(),
	)

	entities, err := query.All(ctx)
	if err != nil {
		return ledgerEntries, err
	}

	resets := []time.Time{}
	for _, entity := range entities {
		reset, err := mapResetEntity(entity)
		if err != nil {
			return ledgerEntries, err
		}

		ledgerEntries.AddReset(reset)
		resets = append(resets, reset.EffectiveAt)
	}
	resets = append(resets, to)

	ledger, err := a.getLedger(ctx, ledgerID)
	if err != nil {
		return ledgerEntries, err
	}

	// fetch the features for each entry
	// TODO: don't fetch all, only what's needed
	allFeatures, err := a.ListFeatures(ctx, credit.ListFeaturesParams{
		Namespace: ledgerID.Namespace,
	})
	if err != nil {
		return ledgerEntries, err
	}

	featureMap := map[credit.FeatureID]credit.Feature{}
	for _, feature := range allFeatures {
		featureMap[*feature.ID] = feature
	}

	meters := map[string]models.Meter{}
	for _, feature := range allFeatures {
		meterSlug := feature.MeterSlug
		if _, ok := meters[meterSlug]; !ok {
			meter, err := a.meterRepository.GetMeterByIDOrSlug(ctx, ledgerID.Namespace, meterSlug)
			if err != nil {
				return ledgerEntries, fmt.Errorf("get meter: %w", err)
			}
			meters[meterSlug] = meter
		}
	}

	balanceFrom := from
	for _, balanceTo := range resets {
		_, entries, err := a.getBalance(ctx, ledgerID, balanceFrom, balanceTo)
		if err != nil {
			return ledgerEntries, err
		}

		if windowParams != nil {
			// for each balance period we get query the windowed usage data
			// from the streaming connector for all usage type entries
			entryList := entries.GetEntries()

			for _, entry := range entryList {
				if entry.Type == credit.LedgerEntryTypeGrantUsage {
					feature, ok := featureMap[*entry.FeatureID]
					if !ok {
						return ledgerEntries, fmt.Errorf("feature not found")
					}

					meter, ok := meters[feature.MeterSlug]
					if !ok {
						return ledgerEntries, fmt.Errorf("meter not found")
					}

					queryParams := streaming.QueryParams{
						From:           &entry.Period.From,
						To:             &entry.Period.To,
						WindowSize:     &windowParams.WindowSize,
						WindowTimeZone: &windowParams.WindowTimeZone,
						FilterSubject:  []string{ledger.Subject},
						Aggregation:    meter.Aggregation,
					}

					if feature.MeterGroupByFilters != nil {
						queryParams.FilterGroupBy = map[string][]string{}
						for k, v := range *feature.MeterGroupByFilters {
							queryParams.FilterGroupBy[k] = []string{v}
						}
					}

					rows, err := a.streamingConnector.QueryMeter(ctx, feature.Namespace, feature.MeterSlug, &queryParams)
					if err != nil {
						return ledgerEntries, err
					}
					if len(rows) == 0 {
						return ledgerEntries, fmt.Errorf("no usage found for meter %s in period %s to %s", feature.MeterSlug, entry.Period.From.Format(time.RFC3339), entry.Period.To.Format(time.RFC3339))
					}

					for _, row := range rows {
						ledgerEntries.AddGrantUsage(entry.ID, entry.FeatureID, row.WindowStart, row.WindowEnd, -row.Value)
					}
				} else {
					ledgerEntries.AddEntry(entry)
				}
			}
		} else {
			ledgerEntries.Append(entries)
		}

		balanceFrom = balanceTo
	}

	// Because of the above we cannot really limit the query from the db side,
	// so we are "emulating" the limit here
	if pagination.Offset > 0 {
		ledgerEntries = ledgerEntries.Skip(pagination.Offset)
	}

	if pagination.Limit > 0 && ledgerEntries.Len() > pagination.Limit {
		ledgerEntries = ledgerEntries.Truncate(pagination.Limit)
	}

	return ledgerEntries, nil
}
