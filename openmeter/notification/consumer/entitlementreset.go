package consumer

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"github.com/openmeterio/openmeter/api"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
	productcatalogdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/driver"
	subjecthttphandler "github.com/openmeterio/openmeter/openmeter/subject/httphandler"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (b *EntitlementSnapshotHandler) isEntitlementResetEvent(event snapshot.SnapshotEvent) bool {
	// If this is not a valid threshold event, it cannot be a reset event
	if !b.isBalanceThresholdEvent(event) {
		return false
	}

	return event.Operation == snapshot.ValueOperationReset
}

func (b *EntitlementSnapshotHandler) handleAsEntitlementResetEvent(ctx context.Context, event snapshot.SnapshotEvent) error {
	affectedRulesPaged, err := b.Notification.ListRules(ctx, notification.ListRulesInput{
		Namespaces: []string{event.Namespace.ID},
		Types:      []notification.EventType{notification.EventTypeEntitlementReset},
	})
	if err != nil {
		return fmt.Errorf("failed to list notification rules: %w", err)
	}

	affectedRules := lo.Filter(affectedRulesPaged.Items, func(rule notification.Rule, _ int) bool {
		if len(rule.Config.EntitlementReset.Features) == 0 {
			return true
		}

		return slices.Contains(rule.Config.EntitlementReset.Features, event.Entitlement.FeatureID) ||
			slices.Contains(rule.Config.EntitlementReset.Features, event.Entitlement.FeatureKey)
	})

	var errs []error

	for _, rule := range affectedRules {
		if !rule.HasEnabledChannels() {
			continue
		}

		if err = b.handleResetRule(ctx, event, rule); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (b *EntitlementSnapshotHandler) handleResetRule(ctx context.Context, event snapshot.SnapshotEvent, rule notification.Rule) error {
	lastEvents, err := b.Notification.ListEvents(ctx, notification.ListEventsInput{
		Page: pagination.Page{
			PageSize:   1,
			PageNumber: 1,
		},
		Namespaces: []string{event.Namespace.ID},

		From: event.Entitlement.CurrentUsagePeriod.From,
		To:   event.Entitlement.CurrentUsagePeriod.To,

		OrderBy: notification.OrderByCreatedAt,
		Order:   sortx.OrderDesc,
	})
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}

	if len(lastEvents.Items) > 0 {
		// We have already created an event for this period, so we don't need to create a new one
		return nil
	}

	return b.createResetEvent(ctx, createEntitlementResetEventInput{
		Snapshot: event,
		RuleID:   rule.ID,
	})
}

type createEntitlementResetEventInput struct {
	Snapshot snapshot.SnapshotEvent
	RuleID   string
}

func (b *EntitlementSnapshotHandler) createResetEvent(ctx context.Context, in createEntitlementResetEventInput) error {
	entitlementAPIEntity, err := entitlementdriver.Parser.ToMetered(&in.Snapshot.Entitlement)
	if err != nil {
		return fmt.Errorf("failed to map entitlement value to API: %w", err)
	}

	annotations := models.Annotations{
		notification.AnnotationEventSubjectKey: in.Snapshot.Subject.Key,
		notification.AnnotationEventFeatureKey: in.Snapshot.Feature.Key,
	}

	if in.Snapshot.Subject.Id != "" {
		annotations[notification.AnnotationEventSubjectID] = in.Snapshot.Subject.Id
	}

	if in.Snapshot.Feature.ID != "" {
		annotations[notification.AnnotationEventFeatureID] = in.Snapshot.Feature.ID
	}

	_, err = b.Notification.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: in.Snapshot.Namespace.ID,
		},
		Annotations: annotations,
		Type:        notification.EventTypeEntitlementReset,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeEntitlementReset,
			},
			EntitlementReset: &notification.EntitlementResetPayload{
				Entitlement: *entitlementAPIEntity,
				Feature:     productcatalogdriver.MapFeatureToResponse(in.Snapshot.Feature),
				Subject:     subjecthttphandler.FromSubject(in.Snapshot.Subject),
				Value:       (api.EntitlementValue)(*in.Snapshot.Value),
			},
		},
		RuleID: in.RuleID,
	})

	return err
}
