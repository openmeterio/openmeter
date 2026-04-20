package lineengine

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SplitLineGroupAdapter interface {
	CreateSplitLineGroup(ctx context.Context, input billing.CreateSplitLineGroupAdapterInput) (billing.SplitLineGroup, error)
	GetSplitLineGroupHeaders(ctx context.Context, input billing.GetSplitLineGroupHeadersInput) (billing.SplitLineGroupHeaders, error)
}

func (e *Engine) SplitGatheringLine(ctx context.Context, in billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	res := billing.SplitGatheringLineResult{}

	if err := in.Validate(); err != nil {
		return res, err
	}

	line := in.Line

	if !line.ServicePeriod.Contains(in.SplitAt) {
		return res, fmt.Errorf("line[%s]: splitAt is not within the line period", line.ID)
	}

	var splitLineGroupID string
	if line.SplitLineGroupID == nil {
		splitLineGroup, err := e.adapter.CreateSplitLineGroup(ctx, billing.CreateSplitLineGroupAdapterInput{
			Namespace: line.Namespace,
			SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
				Name:        line.Name,
				Description: line.Description,
				ServicePeriod: timeutil.ClosedPeriod{
					From: line.ServicePeriod.From,
					To:   line.ServicePeriod.To,
				},
				RatecardDiscounts: line.RateCardDiscounts,
				TaxConfig:         line.TaxConfig,
			},
			UniqueReferenceID: line.ChildUniqueReferenceID,
			Currency:          line.Currency,
			Price:             lo.ToPtr(line.Price),
			FeatureKey:        lo.EmptyableToPtr(line.FeatureKey),
			Subscription:      line.Subscription,
		})
		if err != nil {
			return res, fmt.Errorf("creating split line group: %w", err)
		}

		splitLineGroupID = splitLineGroup.ID
	} else {
		splitLineGroupID = lo.FromPtr(line.SplitLineGroupID)
		if splitLineGroupID == "" {
			return res, fmt.Errorf("split line group id is empty")
		}
	}

	postSplitAtLine, err := line.CloneForCreate(func(l *billing.GatheringLine) {
		l.ServicePeriod.From = in.SplitAt
		l.SplitLineGroupID = lo.ToPtr(splitLineGroupID)
		l.ChildUniqueReferenceID = nil
	})
	if err != nil {
		return res, fmt.Errorf("cloning post split line: %w", err)
	}

	postSplitAtLineEmpty, err := isPeriodEmptyConsideringTruncations(postSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if post split line is empty: %w", err)
	}

	if !postSplitAtLineEmpty {
		if err := postSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating post split line: %w", err)
		}
	}

	line.ServicePeriod.To = in.SplitAt
	line.InvoiceAt = in.SplitAt
	line.SplitLineGroupID = lo.ToPtr(splitLineGroupID)
	line.ChildUniqueReferenceID = nil

	preSplitAtLine := line

	preSplitAtLineEmpty, err := isPeriodEmptyConsideringTruncations(preSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if pre split line is empty: %w", err)
	}

	if preSplitAtLineEmpty {
		preSplitAtLine.DeletedAt = lo.ToPtr(clock.Now())
	} else {
		if err := preSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating pre split line: %w", err)
		}
	}

	var postSplitAtLinePtr *billing.GatheringLine
	if !postSplitAtLineEmpty {
		postSplitAtLinePtr = &postSplitAtLine
	}

	return billing.SplitGatheringLineResult{
		PreSplitAtLine:  preSplitAtLine,
		PostSplitAtLine: postSplitAtLinePtr,
	}, nil
}

func (e *Engine) ResolveSplitLineGroupHeaders(ctx context.Context, ns string, lines billing.StandardLines) error {
	splitLineGroupIDs := lo.Uniq(
		lo.Filter(
			lo.Map(lines, func(line *billing.StandardLine, _ int) string { return lo.FromPtr(line.SplitLineGroupID) }),
			func(id string, _ int) bool { return id != "" },
		),
	)

	if len(splitLineGroupIDs) == 0 {
		return nil
	}

	splitLineGroupHeaders, err := e.adapter.GetSplitLineGroupHeaders(ctx, billing.GetSplitLineGroupHeadersInput{
		Namespace:         ns,
		SplitLineGroupIDs: splitLineGroupIDs,
	})
	if err != nil {
		return fmt.Errorf("getting split line group headers: %w", err)
	}

	splitLineGroupHeadersByID := lo.SliceToMap(splitLineGroupHeaders, func(header billing.SplitLineGroup) (string, billing.SplitLineGroup) {
		return header.ID, header
	})

	for idx := range lines {
		if lines[idx].SplitLineGroupID == nil {
			continue
		}

		splitLineGroupHeader, ok := splitLineGroupHeadersByID[lo.FromPtr(lines[idx].SplitLineGroupID)]
		if !ok {
			return fmt.Errorf("split line group header not found for line[%s]: id[%s]", lines[idx].ID, lo.FromPtr(lines[idx].SplitLineGroupID))
		}

		lines[idx].SplitLineHierarchy = &billing.SplitLineHierarchy{
			Group: splitLineGroupHeader,
		}
	}

	return nil
}

func isPeriodEmptyConsideringTruncations(line billing.GatheringLine) (bool, error) {
	price := line.GetPrice()
	if price == nil {
		return false, fmt.Errorf("price is nil")
	}

	if price.Type() == productcatalog.FlatPriceType {
		return false, nil
	}

	return line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty(), nil
}
