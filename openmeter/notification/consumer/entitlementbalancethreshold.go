package consumer

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"time"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
	productcatalogdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/driver"
	subjecthttphandler "github.com/openmeterio/openmeter/openmeter/subject/httphandler"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type EntitlementSnapshotHandlerState struct {
	TotalGrants float64 `json:"totalGrants"`
}

var ErrNoBalanceAvailable = errors.New("no balance available")

func (b *EntitlementSnapshotHandler) handleAsSnapshotEvent(ctx context.Context, event snapshot.SnapshotEvent) error {
	// TODO[issue-1364]: this must be cached to prevent going to the DB for each balance.snapshot event
	affectedRulesPaged, err := b.Notification.ListRules(ctx, notification.ListRulesInput{
		Namespaces: []string{event.Namespace.ID},
		Types:      []notification.EventType{notification.EventTypeBalanceThreshold},
	})
	if err != nil {
		return fmt.Errorf("failed to list notification rules: %w", err)
	}

	affectedRules := lo.Filter(affectedRulesPaged.Items, func(rule notification.Rule, _ int) bool {
		if len(rule.Config.BalanceThreshold.Thresholds) == 0 {
			return false
		}

		if len(rule.Config.BalanceThreshold.Features) == 0 {
			return true
		}

		return slices.Contains(rule.Config.BalanceThreshold.Features, event.Entitlement.FeatureID) ||
			slices.Contains(rule.Config.BalanceThreshold.Features, event.Entitlement.FeatureKey)
	})

	var errs []error

	for _, rule := range affectedRules {
		if !rule.HasEnabledChannels() {
			continue
		}

		if err = b.handleRule(ctx, event, rule); err != nil {
			errs = append(
				errs,
				fmt.Errorf("failed to handle event for rule [namespace=%s notification_rule.id=%s entitlement.id=%s]: %w", rule.Namespace, rule.ID, event.Entitlement.ID, err),
			)
		}
	}

	return errors.Join(errs...)
}

func (b *EntitlementSnapshotHandler) handleRule(ctx context.Context, balSnapshot snapshot.SnapshotEvent, rule notification.Rule) error {
	// Check 1: do we have a threshold we should create an event for?

	thresholds, err := getActiveThresholdsWithHighestPriority(rule.Config.BalanceThreshold.Thresholds, *balSnapshot.Value)
	if err != nil {
		return fmt.Errorf("failed to calculate active thresholds: %w", err)
	}

	for _, threshold := range thresholds.Iter() {
		if threshold == nil {
			// Skip nil thresholds as there might be scenarios where only usage or balance thresholds are being active.
			continue
		}

		// Check 2: fetch the last event for the same period and validate if we need to send a new notification

		periodDedupeHash := b.getPeriodsDeduplicationHash(balSnapshot, rule.ID)

		// TODO[issue-1364]: this must be cached to prevent going to the DB for each balance.snapshot event
		lastEvents, err := b.Notification.ListEvents(ctx, notification.ListEventsInput{
			Page: pagination.Page{
				PageSize:   1,
				PageNumber: 1,
			},
			Namespaces: []string{balSnapshot.Namespace.ID},

			From: balSnapshot.Entitlement.CurrentUsagePeriod.From,
			To:   balSnapshot.Entitlement.CurrentUsagePeriod.To,

			DeduplicationHashes: []string{periodDedupeHash},
			OrderBy:             notification.OrderByCreatedAt,
			Order:               sortx.OrderDesc,
		})
		if err != nil {
			return fmt.Errorf("failed to list events [dedup.key=%s]: %w", periodDedupeHash, err)
		}

		createEventInput := createBalanceThresholdEventInput{
			Snapshot:   balSnapshot,
			DedupeHash: periodDedupeHash,
			Threshold:  *threshold,
			RuleID:     rule.ID,
		}

		if len(lastEvents.Items) == 0 {
			// we need to trigger the event, as we have hit a threshold and have no previous event
			err = b.createEvent(ctx, createEventInput)
			if err != nil {
				return fmt.Errorf("failed to create event: %w", err)
			}

			continue
		}

		lastEvent := lastEvents.Items[0]

		if lastEvent.Payload.Type != notification.EventTypeBalanceThreshold {
			// This should never happen, but let's log it and trigger the event, so that we have a better reference point
			// in place
			b.Logger.ErrorContext(ctx, "last event is not a balance threshold event", slog.String("event.id", lastEvent.ID))

			err = b.createEvent(ctx, createEventInput)
			if err != nil {
				return fmt.Errorf("failed to create event: %w", err)
			}

			continue
		}

		lastEventActualValue, err := getNumericThreshold(
			lastEvent.Payload.BalanceThreshold.Threshold,
			lastEvent.Payload.BalanceThreshold.Value)
		if err != nil {
			if errors.Is(err, ErrNoBalanceAvailable) {
				// In case there are no grants, percentage all percentage rules would match, so let's instead
				// wait until we have some credits to calculate the actual value
				b.Logger.Warn("no balance available skipping event creation", "last_event_id", lastEvent.ID)

				continue
			}

			return fmt.Errorf("failed to calculate actual value from last event: %w", err)
		}

		if lastEventActualValue.BalanceThreshold != *threshold {
			// The last event was triggered by a different threshold, so we need to trigger a new event
			err = b.createEvent(ctx, createEventInput)
			if err != nil {
				return fmt.Errorf("failed to create event: %w", err)
			}
		}
	}

	return nil
}

type createBalanceThresholdEventInput struct {
	Snapshot   snapshot.SnapshotEvent
	DedupeHash string
	Threshold  notification.BalanceThreshold
	RuleID     string
}

func (b *EntitlementSnapshotHandler) createEvent(ctx context.Context, in createBalanceThresholdEventInput) error {
	entitlementAPIEntity, err := entitlementdriver.Parser.ToMetered(&in.Snapshot.Entitlement)
	if err != nil {
		return fmt.Errorf("failed to map entitlement value to API: %w", err)
	}

	annotations := models.Annotations{
		notification.AnnotationEventSubjectKey:        in.Snapshot.Subject.Key,
		notification.AnnotationEventFeatureKey:        in.Snapshot.Feature.Key,
		notification.AnnotationBalanceEventDedupeHash: in.DedupeHash,
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
		Type:        notification.EventTypeBalanceThreshold,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type: notification.EventTypeBalanceThreshold,
			},
			BalanceThreshold: &notification.BalanceThresholdPayload{
				EntitlementValuePayloadBase: notification.EntitlementValuePayloadBase{
					Entitlement: *entitlementAPIEntity,
					Feature:     productcatalogdriver.MapFeatureToResponse(in.Snapshot.Feature),
					Subject:     subjecthttphandler.FromSubject(in.Snapshot.Subject),
					Value:       (api.EntitlementValue)(*in.Snapshot.Value),
				},
				Threshold: in.Threshold,
			},
		},
		RuleID:                   in.RuleID,
		HandlerDeduplicationHash: in.DedupeHash,
	})

	return err
}

func (b *EntitlementSnapshotHandler) isBalanceThresholdEvent(event snapshot.SnapshotEvent) bool {
	if event.Entitlement.EntitlementType != entitlement.EntitlementTypeMetered {
		return false
	}

	// We don't care about delete events, but reset events are valid snapshot events
	if !slices.Contains([]snapshot.ValueOperationType{snapshot.ValueOperationUpdate, snapshot.ValueOperationReset}, event.Operation) {
		return false
	}

	// We don't care about events of inactive entitlements
	if !event.Entitlement.IsActive(clock.Now()) {
		return false
	}

	// Let's validate the event value contains all the necessary fields for calculations
	if event.Value == nil || event.Value.Balance == nil || event.Value.Usage == nil {
		return false
	}

	return true
}

// getPeriodsDeduplicationHash generates a hash that the handler can use to deduplicate the events. Right now the hash is unique
// for a single entitlement usage period. We can use this to fetch the previous events for the same period and validate
// if we need to send a new notification.
func (b *EntitlementSnapshotHandler) getPeriodsDeduplicationHash(snapshot snapshot.SnapshotEvent, ruleID string) string {
	// Note: this should not happen, but let's be safe here
	currentUsagePeriod := lo.FromPtrOr(
		snapshot.Entitlement.CurrentUsagePeriod, timeutil.ClosedPeriod{
			From: time.Time{},
			To:   time.Time{},
		})

	source := strings.Join([]string{
		ruleID,
		snapshot.Namespace.ID,
		currentUsagePeriod.From.UTC().Format(time.RFC3339),
		currentUsagePeriod.To.UTC().Format(time.RFC3339),
		snapshot.Subject.Key,
		snapshot.Entitlement.ID,
		snapshot.Feature.ID,
		lo.FromPtrOr(snapshot.Entitlement.MeasureUsageFrom, time.Time{}).UTC().Format(time.RFC3339),
	}, "/")

	h := sha256.New()

	h.Write([]byte(source))

	bs := h.Sum(nil)

	// bsnap == balance.snapshot
	// v1 == version 1 (in case we need to change the hashing strategy)
	return fmt.Sprintf("bsnap_v1_%x", bs)
}

type numericThreshold struct {
	notification.BalanceThreshold

	// ThresholdValue always contains the credit value of the threshold regardless if it's a percentage
	// or a value threshold
	ThresholdValue float64

	// Active is true if the threshold value has been reached, otherwise it is false.
	Active bool
}

func getNumericThreshold(threshold notification.BalanceThreshold, value api.EntitlementValue) (*numericThreshold, error) {
	switch threshold.Type {
	// Deprecated: obsoleted by api.NotificationRuleBalanceThresholdValueTypeUsageValue
	case api.NotificationRuleBalanceThresholdValueTypeNumber:
		fallthrough
	case api.NotificationRuleBalanceThresholdValueTypeUsageValue:
		return &numericThreshold{
			BalanceThreshold: threshold,
			ThresholdValue:   threshold.Value,
			Active:           threshold.Value <= lo.FromPtr(value.Usage),
		}, nil
	// Deprecated: obsoleted by api.NotificationRuleBalanceThresholdValueTypeUsagePercentage
	case api.NotificationRuleBalanceThresholdValueTypePercent:
		fallthrough
	case api.NotificationRuleBalanceThresholdValueTypeUsagePercentage:
		totalGrants := lo.FromPtr(value.Balance) + lo.FromPtr(value.Usage) - lo.FromPtr(value.Overage)

		// In case there are no grants yet, we can't calculate the actual value, we are filtering out the
		// thresholds to prevent event triggering in the following scenario:
		//
		// - A new entitlement is created (and there are balance threshold rules active)
		// - Then the granting is done as a separate step
		//
		// As this would mean that we would trigger a notification for the first activity for 100%
		if totalGrants == 0 {
			return nil, ErrNoBalanceAvailable
		}

		thresholdValue := totalGrants * threshold.Value / 100

		return &numericThreshold{
			BalanceThreshold: threshold,
			ThresholdValue:   thresholdValue,
			Active:           thresholdValue <= lo.FromPtr(value.Usage),
		}, nil
	case api.NotificationRuleBalanceThresholdValueTypeBalanceValue:
		return &numericThreshold{
			BalanceThreshold: threshold,
			ThresholdValue:   threshold.Value,
			Active:           threshold.Value >= lo.FromPtr(value.Balance),
		}, nil
	default:
		return nil, errors.New("unknown threshold type")
	}
}

type activeThresholds struct {
	Usage   *notification.BalanceThreshold
	Balance *notification.BalanceThreshold
}

func (a activeThresholds) Iter() iter.Seq2[int, *notification.BalanceThreshold] {
	thresholds := []*notification.BalanceThreshold{a.Usage, a.Balance}

	return func(yield func(int, *notification.BalanceThreshold) bool) {
		for i := 0; i <= len(thresholds)-1; i++ {
			if !yield(i, thresholds[i]) {
				return
			}
		}
	}
}

func getActiveThresholdsWithHighestPriority(thresholds []notification.BalanceThreshold, value snapshot.EntitlementValue) (*activeThresholds, error) {
	var (
		usage   *numericThreshold
		balance *numericThreshold
	)

	for _, threshold := range thresholds {
		numThreshold, err := getNumericThreshold(threshold, api.EntitlementValue(value))
		if err != nil {
			if errors.Is(err, ErrNoBalanceAvailable) {
				continue
			}

			return nil, err
		}

		// Skip non-active thresholds
		if !numThreshold.Active {
			continue
		}

		switch numThreshold.BalanceThreshold.Type {
		case api.NotificationRuleBalanceThresholdValueTypeBalanceValue:
			if balance == nil {
				balance = numThreshold
			} else if balance.ThresholdValue > numThreshold.ThresholdValue {
				balance = numThreshold
			}
		// Deprecated: obsoleted by api.NotificationRuleBalanceThresholdValueTypeUsagePercentage
		case api.NotificationRuleBalanceThresholdValueTypePercent:
			fallthrough
		case api.NotificationRuleBalanceThresholdValueTypeUsagePercentage:
			if usage == nil {
				usage = numThreshold
			} else if usage.ThresholdValue <= numThreshold.ThresholdValue {
				usage = numThreshold
			}
		// Deprecated: obsoleted by api.NotificationRuleBalanceThresholdValueTypeUsageValue
		case api.NotificationRuleBalanceThresholdValueTypeNumber:
			fallthrough
		case api.NotificationRuleBalanceThresholdValueTypeUsageValue:
			if usage == nil {
				usage = numThreshold
			} else if usage.ThresholdValue < numThreshold.ThresholdValue {
				usage = numThreshold
			}
		default:
			return nil, fmt.Errorf("unknown balance threshold type: %s", numThreshold.BalanceThreshold.Type)
		}
	}

	result := &activeThresholds{}

	if usage != nil {
		result.Usage = lo.ToPtr(usage.BalanceThreshold)
	}

	if balance != nil {
		result.Balance = lo.ToPtr(balance.BalanceThreshold)
	}

	return result, nil
}
