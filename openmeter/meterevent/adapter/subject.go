package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// listSubjectsMaxScanRounds bounds how many streaming batches a single request
// scans when the attributed filter discards most of a batch. When the cap is
// hit, the result carries a next cursor so the client can continue scanning
// even though the page is not full.
const listSubjectsMaxScanRounds = 10

// ListSubjects returns the subjects of the ingested events ordered by key.
//
// The attributed filter is evaluated after resolving customer usage
// attribution, so the page is assembled by scanning streaming batches until it
// fills up, the subjects are exhausted, or the scan round cap is hit. A page
// shorter than the requested limit therefore does NOT imply the listing is
// exhausted; only an absent next cursor does.
func (a *adapter) ListSubjects(ctx context.Context, params meterevent.ListSubjectsParams) (pagination.Result[meterevent.Subject], error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return pagination.Result[meterevent.Subject]{}, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	limit := lo.FromPtrOr(params.Limit, meterevent.MaximumLimit)

	// Scanning one key past the limit proves whether more data follows a full
	// page, so the next cursor can be decided without another batch. With the
	// attributed filter, full-size batches fill the page in fewer rounds.
	scanLimit := limit + 1
	if params.Attributed != nil {
		scanLimit = meterevent.MaximumLimit
	}

	result := pagination.Result[meterevent.Subject]{Items: make([]meterevent.Subject, 0, limit)}
	cursor := params.Cursor

	for range listSubjectsMaxScanRounds {
		// Get the next batch of subject keys
		keys, err := a.streamingConnector.ListSubjectsV2(ctx, streaming.ListSubjectsV2Params{
			Namespace: params.Namespace,
			Key:       params.Key,
			Cursor:    cursor,
			Limit:     &scanLimit,
		})
		if err != nil {
			return pagination.Result[meterevent.Subject]{}, fmt.Errorf("query subjects: %w", err)
		}

		// Attribution is only resolved when the attributed filter needs it.
		var attributedKeys map[string]struct{}
		if params.Attributed != nil {
			attributedKeys, err = a.attributedSubjectKeys(ctx, params.Namespace, keys)
			if err != nil {
				return pagination.Result[meterevent.Subject]{}, fmt.Errorf("resolve attributed subjects: %w", err)
			}
		}

		for _, key := range keys {
			if params.Attributed != nil && *params.Attributed != lo.HasKey(attributedKeys, key) {
				continue
			}

			result.Items = append(result.Items, meterevent.Subject{Key: key})
		}

		// A match beyond the limit proves more data follows the trimmed page.
		if len(result.Items) > limit {
			result.Items = result.Items[:limit]
			result.NextCursor = lo.ToPtr(result.Items[limit-1].Cursor())

			return result, nil
		}

		// The page is exactly full but the batch may not have been the last
		// one: stop scanning and resume from the last scanned key instead of
		// fetching more batches just to prove whether a next page exists. The
		// continuation may turn out to be empty, which the contract permits
		// (only an absent next cursor signals exhaustion).
		if len(result.Items) == limit && len(keys) == scanLimit {
			result.NextCursor = lo.ToPtr(meterevent.Subject{Key: keys[len(keys)-1]}.Cursor())

			return result, nil
		}

		// The batch was the last one; whatever matched is the final page.
		if len(keys) < scanLimit {
			return result, nil
		}

		cursor = lo.ToPtr(meterevent.Subject{Key: keys[len(keys)-1]}.Cursor())
	}

	// Scan round cap hit: resume from the last scanned subject key.
	result.NextCursor = cursor

	return result, nil
}

// attributedSubjectKeys resolves which subject keys are attributed to a customer.
// Usage attribution matches a subject key against either the customer's usage
// attribution subject keys or the customer's own key.
func (a *adapter) attributedSubjectKeys(ctx context.Context, namespace string, keys []string) (map[string]struct{}, error) {
	attributed := map[string]struct{}{}

	// The streaming query excludes empty subjects, but rows from direct
	// producers must not reach the customer filters: pkg/filter rejects empty
	// values, which would fail the whole listing.
	keys = lo.Filter(keys, func(key string, _ int) bool { return key != "" })

	if len(keys) == 0 {
		return attributed, nil
	}

	for _, input := range []customer.ListCustomersInput{
		{
			Namespace:                  namespace,
			UsageAttributionSubjectKey: &filter.FilterString{In: &keys},
		},
		{
			Namespace: namespace,
			Key:       &filter.FilterString{In: &keys},
		},
	} {
		customerList, err := a.customerService.ListCustomers(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list customers: %w", err)
		}

		for _, c := range customerList.Items {
			for _, value := range c.GetUsageAttribution().GetValues() {
				attributed[value] = struct{}{}
			}
		}
	}

	return attributed, nil
}
