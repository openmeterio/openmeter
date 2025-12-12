package meterexportservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type funnelParams struct {
	meter       meter.Meter
	queryParams streaming.QueryParams
}

func (p funnelParams) validate() error {
	var errs []error

	if err := p.queryParams.Validate(); err != nil {
		errs = append(errs, err)
	}

	if p.queryParams.From == nil {
		errs = append(errs, errors.New("query params from is required"))
	}

	if p.queryParams.WindowSize == nil {
		errs = append(errs, errors.New("query params window size is required"))
	}

	if unsupportedErrs := p.validateUnsupportedParams(); len(unsupportedErrs) > 0 {
		errs = append(errs, unsupportedErrs...)
	}

	return errors.Join(errs...)
}

// TODO: we'll later support these
func (p funnelParams) validateUnsupportedParams() []error {
	var errs []error

	if p.queryParams.ClientID != nil {
		errs = append(errs, errors.New("client id is not supported"))
	}

	if len(p.queryParams.FilterCustomer) > 0 {
		errs = append(errs, errors.New("filter customer is not supported"))
	}

	if len(p.queryParams.FilterGroupBy) > 0 {
		errs = append(errs, errors.New("filter group by is not supported"))
	}

	// GroupBy subject is allowed (used internally for per-subject export)
	for _, g := range p.queryParams.GroupBy {
		if g != "subject" {
			errs = append(errs, errors.New("group by is only supported for subject"))
			break
		}
	}

	return errs
}

const TARGET_ROWS_PER_QUERY = 500

// funnel reads the calculated meter values to pass them on to later stages.
// funnel is a streaming operation and only returns an error if the operation fails to start.
func (s *service) funnel(ctx context.Context, params funnelParams, resultCh chan<- meter.MeterQueryRow, errCh chan<- error) error {
	defer func() {
		close(resultCh)
		close(errCh)
	}()

	if err := params.validate(); err != nil {
		return fmt.Errorf("validate params: %w", err)
	}

	// We'll keep querying the meter in given intervals which we determine by the number of rows returned by the query.
	queryFrom := *params.queryParams.From

	queryTo, err := iterateQueryTime(queryFrom, params.queryParams.To, *params.queryParams.WindowSize)
	if err != nil {
		return fmt.Errorf("calculate query to: %w", err)
	}

	for {
		if ctx.Err() != nil {
			errCh <- ctx.Err()
			return nil
		}

		if !queryTo.After(queryFrom) {
			break
		}

		queryParams := streaming.QueryParams{
			From:           &queryFrom,
			To:             &queryTo,
			WindowSize:     params.queryParams.WindowSize,
			WindowTimeZone: params.queryParams.WindowTimeZone,
		}

		rows, err := s.StreamingConnector.QueryMeter(ctx, params.meter.Namespace, params.meter, queryParams)
		if err != nil {
			errCh <- fmt.Errorf("query meter: %w", err)
			break
		}

		for _, row := range rows {
			resultCh <- row
		}

		// Let's update the query from and to values
		nextQueryTo, err := iterateQueryTime(queryTo, params.queryParams.To, *params.queryParams.WindowSize)
		if err != nil {
			return fmt.Errorf("calculate next query to: %w", err)
		}

		queryFrom = queryTo
		queryTo = nextQueryTo
	}

	return nil
}

func iterateQueryTime(start time.Time, limit *time.Time, windowSize meter.WindowSize) (time.Time, error) {
	out := start
	var err error

	for i := 0; i < TARGET_ROWS_PER_QUERY; i++ {
		out, err = windowSize.AddTo(out)
		if err != nil {
			return time.Time{}, fmt.Errorf("add to: %w", err)
		}

		if limit != nil && out.After(*limit) {
			out = *limit
			break
		}
	}

	return out, nil
}
