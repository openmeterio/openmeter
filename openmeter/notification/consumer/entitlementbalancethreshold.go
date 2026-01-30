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
	"github.com/zeebo/xxh3"
	"golang.org/x/exp/slices"

	"github.com/openmeterio/openmeter/api"
	customerhttphandler "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
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

		dedupHash, err := NewBalanceEventDedupHash(balSnapshot, rule.ID, *threshold)
		if err != nil {
			return fmt.Errorf("failed to generate deduplication hash: %w", err)
		}

		// TODO[issue-1364]: this must be cached to prevent going to the DB for each balance.snapshot event
		lastEvents, err := b.Notification.ListEvents(ctx, notification.ListEventsInput{
			Page: pagination.Page{
				PageSize:   1,
				PageNumber: 1,
			},
			Namespaces: []string{balSnapshot.Namespace.ID},

			From: balSnapshot.Entitlement.CurrentUsagePeriod.From,
			To:   balSnapshot.Entitlement.CurrentUsagePeriod.To,

			DeduplicationHashes: []string{dedupHash.V1(), dedupHash.V2()},
			OrderBy:             notification.OrderByCreatedAt,
			Order:               sortx.OrderDesc,
		})
		if err != nil {
			return fmt.Errorf("failed to list events [dedup.hash.v1=%s dedup.hash.v2=%s]: %w", dedupHash.V1(), dedupHash.V2(), err)
		}

		createEventInput := createBalanceThresholdEventInput{
			Snapshot:   balSnapshot,
			DedupeHash: dedupHash.V2(),
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
				b.Logger.WarnContext(ctx, "no balance available skipping event creation", "last_event_id", lastEvent.ID)

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
	entitlementAPIEntity, err := entitlementdriver.Parser.ToMetered(&entitlement.EntitlementWithCustomer{Entitlement: in.Snapshot.Entitlement, Customer: in.Snapshot.Customer})
	if err != nil {
		return fmt.Errorf("failed to map entitlement value to API: %w", err)
	}

	annotations := models.Annotations{
		notification.AnnotationEventSubjectKey:        in.Snapshot.Subject.Key,
		notification.AnnotationEventCustomerID:        in.Snapshot.Customer.ID,
		notification.AnnotationEventFeatureKey:        in.Snapshot.Feature.Key,
		notification.AnnotationBalanceEventDedupeHash: in.DedupeHash,
	}

	if in.Snapshot.Subject.Id != "" {
		annotations[notification.AnnotationEventSubjectID] = in.Snapshot.Subject.Id
	}

	if in.Snapshot.Customer.Key != nil {
		annotations[notification.AnnotationEventCustomerKey] = *in.Snapshot.Customer.Key
	}

	if in.Snapshot.Feature.ID != "" {
		annotations[notification.AnnotationEventFeatureID] = in.Snapshot.Feature.ID
	}

	valueBase := notification.EntitlementValuePayloadBase{
		Entitlement: *entitlementAPIEntity,
		Feature:     productcatalogdriver.MapFeatureToResponse(in.Snapshot.Feature),
		Subject:     subjecthttphandler.FromSubject(in.Snapshot.Subject),
		Value:       (api.EntitlementValue)(*in.Snapshot.Value),
	}

	// TODO(OM-1508): As we're reusing the same event version, we need to add this temporary check
	if in.Snapshot.Customer.ID != "" {
		apiCustomer, err := customerhttphandler.CustomerToAPI(in.Snapshot.Customer, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to map customer to API: %w", err)
		}

		valueBase.Customer = apiCustomer
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
				EntitlementValuePayloadBase: valueBase,
				Threshold:                   in.Threshold,
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

type ThresholdKind string

const (
	ThresholdKindUsageThreshold   ThresholdKind = "usage"
	ThresholdKindBalanceThreshold ThresholdKind = "balance"
)

func thresholdKindFromThreshold(t notification.BalanceThreshold) (ThresholdKind, error) {
	switch t.Type {
	case api.NotificationRuleBalanceThresholdValueTypeBalanceValue:
		return ThresholdKindBalanceThreshold, nil
	case api.NotificationRuleBalanceThresholdValueTypePercent, api.NotificationRuleBalanceThresholdValueTypeNumber:
		fallthrough
	case api.NotificationRuleBalanceThresholdValueTypeUsagePercentage, api.NotificationRuleBalanceThresholdValueTypeUsageValue:
		return ThresholdKindUsageThreshold, nil
	default:
		return "", fmt.Errorf("unknown threshold type: %s", t.Type)
	}
}

type BalanceEventDedupHash struct {
	currentUsagePeriodFrom time.Time
	currentUsagePeriodTo   time.Time
	notificationRuleID     string
	thresholdKind          ThresholdKind
	namespaceID            string
	subjectKey             string
	entitlementID          string
	featureID              string
	measureUsageFrom       time.Time

	v1 *string
	v2 *string
}

func NewBalanceEventDedupHash(snapshot snapshot.SnapshotEvent, ruleID string, threshold notification.BalanceThreshold) (*BalanceEventDedupHash, error) {
	currentUsagePeriod := lo.FromPtrOr(
		snapshot.Entitlement.CurrentUsagePeriod, timeutil.ClosedPeriod{
			From: time.Time{},
			To:   time.Time{},
		})

	thresholdKind, err := thresholdKindFromThreshold(threshold)
	if err != nil {
		return nil, err
	}

	return &BalanceEventDedupHash{
		currentUsagePeriodFrom: currentUsagePeriod.From,
		currentUsagePeriodTo:   currentUsagePeriod.To,
		notificationRuleID:     ruleID,
		thresholdKind:          thresholdKind,
		namespaceID:            snapshot.Namespace.ID,
		subjectKey:             snapshot.Subject.Key,
		entitlementID:          snapshot.Entitlement.ID,
		featureID:              snapshot.Feature.ID,
		measureUsageFrom:       lo.FromPtrOr(snapshot.Entitlement.MeasureUsageFrom, time.Time{}),
	}, nil
}

func (d BalanceEventDedupHash) V1() string {
	if d.v1 != nil {
		return *d.v1
	}

	source := strings.Join([]string{
		d.notificationRuleID,
		d.namespaceID,
		d.currentUsagePeriodFrom.UTC().Format(time.RFC3339),
		d.currentUsagePeriodTo.UTC().Format(time.RFC3339),
		d.subjectKey,
		d.entitlementID,
		d.featureID,
		d.measureUsageFrom.UTC().Format(time.RFC3339),
	}, "/")

	h := sha256.New()

	h.Write([]byte(source))

	// bsnap == balance.snapshot
	// v1 == version 1 (in case we need to change the hashing strategy)
	d.v1 = lo.ToPtr(fmt.Sprintf("bsnap_v1_%x", h.Sum(nil)))

	return *d.v1
}

func (d BalanceEventDedupHash) V2() string {
	if d.v2 != nil {
		return *d.v2
	}

	source := strings.Join([]string{
		string(d.thresholdKind),
		d.notificationRuleID,
		d.namespaceID,
		d.currentUsagePeriodFrom.UTC().Format(time.RFC3339),
		d.currentUsagePeriodTo.UTC().Format(time.RFC3339),
		d.subjectKey,
		d.entitlementID,
		d.featureID,
		d.measureUsageFrom.UTC().Format(time.RFC3339),
	}, "")

	d.v2 = lo.ToPtr(fmt.Sprintf("bsnap_v2_%x", xxh3.HashString128(source).Bytes()))

	return *d.v2
}

type numericThreshold struct {
	notification.BalanceThreshold

	// ThresholdValue always contains the credit value of the threshold regardless if it's a percentage
	// or a value threshold
	ThresholdValue float64

	// Active is true if the threshold value has been reached, otherwise it is false.
	Active bool
}

const absoluteZero = 1e-9

func getNumericThreshold(threshold notification.BalanceThreshold, value api.EntitlementValue) (*numericThreshold, error) {
	var (
		balance = lo.FromPtr(value.Balance)
		usage   = lo.FromPtr(value.Usage)
		overage = lo.FromPtr(value.Overage)
	)

	// Invalid entitlement value as there cannot be overage if the balance is not zero.
	if balance > absoluteZero && overage > absoluteZero {
		return nil, errors.New("balance and overage cannot be positive number at the same time")
	}

	switch threshold.Type {
	// Deprecated: obsoleted by api.NotificationRuleBalanceThresholdValueTypeUsageValue
	case api.NotificationRuleBalanceThresholdValueTypeNumber:
		fallthrough
	case api.NotificationRuleBalanceThresholdValueTypeUsageValue:
		return &numericThreshold{
			BalanceThreshold: threshold,
			ThresholdValue:   threshold.Value,
			Active:           threshold.Value < usage,
		}, nil
	// Deprecated: obsoleted by api.NotificationRuleBalanceThresholdValueTypeUsagePercentage
	case api.NotificationRuleBalanceThresholdValueTypePercent:
		fallthrough
	case api.NotificationRuleBalanceThresholdValueTypeUsagePercentage:
		// Cannot calculate total grants if both balance and overage are zero (usage alone is insufficient).
		if balance == 0 && overage == 0 {
			return nil, ErrNoBalanceAvailable
		}

		thresholdValue := (balance + usage - overage) * (threshold.Value / 100)

		return &numericThreshold{
			BalanceThreshold: threshold,
			ThresholdValue:   thresholdValue,
			Active:           thresholdValue < usage,
		}, nil
	case api.NotificationRuleBalanceThresholdValueTypeBalanceValue:
		// Cannot calculate total grants if both the balance and the overage have zero value even if the usage is non-zero.
		if balance == 0 && usage == 0 && overage == 0 {
			return nil, ErrNoBalanceAvailable
		}

		active := threshold.Value > balance

		if threshold.Value == 0 {
			active = threshold.Value >= balance
		}

		return &numericThreshold{
			BalanceThreshold: threshold,
			ThresholdValue:   threshold.Value,
			Active:           active,
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
