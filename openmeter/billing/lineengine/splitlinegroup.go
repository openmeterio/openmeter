package lineengine

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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
		splitLineGroup, err := e.splitLineGroupService.CreateSplitLineGroup(ctx, splitlinegroup.SplitLineGroupCreate{
			Namespace: line.Namespace,
			SplitLineGroupMutableFields: splitlinegroup.SplitLineGroupMutableFields{
				Name:        line.Name,
				Description: line.Description,
				ServicePeriod: timeutil.ClosedPeriod{
					From: line.ServicePeriod.From,
					To:   line.ServicePeriod.To,
				},
				RatecardDiscounts: line.RateCardDiscounts,
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

func splitLineGroupIDs(lines billing.StandardLines) []string {
	return lo.Uniq(lo.FilterMap(lines, func(line *billing.StandardLine, _ int) (string, bool) {
		if line == nil || line.SplitLineGroupID == nil {
			return "", false
		}

		return *line.SplitLineGroupID, *line.SplitLineGroupID != ""
	}))
}

func (e *Engine) resolveSplitLineHierarchies(ctx context.Context, ns string, lines billing.StandardLines) (linesWithHierarchy, error) {
	resolvedLines := lo.Map(lines, func(line *billing.StandardLine, _ int) lineWithHierarchy {
		return lineWithHierarchy{StandardLine: line}
	})
	groupIDs := splitLineGroupIDs(lines)
	if len(groupIDs) == 0 {
		return resolvedLines, nil
	}

	hierarchies, err := e.splitLineGroupService.GetSplitLineGroups(ctx, splitlinegroup.GetSplitLineGroupsInput{
		Namespace:         ns,
		SplitLineGroupIDs: groupIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("getting split line groups: %w", err)
	}

	hierarchiesByID := lo.SliceToMap(hierarchies, func(hierarchy splitlinegroup.SplitLineHierarchy) (models.NamespacedID, *splitlinegroup.SplitLineHierarchy) {
		return hierarchy.Group.NamespacedID, &hierarchy
	})

	for idx, line := range resolvedLines {
		if line.SplitLineGroupID == nil {
			continue
		}

		hierarchy := hierarchiesByID[models.NamespacedID{Namespace: line.Namespace, ID: *line.SplitLineGroupID}]
		if hierarchy == nil {
			return nil, fmt.Errorf("split line group[%s] for line[%s] not found", *line.SplitLineGroupID, line.ID)
		}

		resolvedLines[idx].hierarchy = hierarchy
	}

	return resolvedLines, nil
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
