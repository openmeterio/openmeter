// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumer

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
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
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type BalanceThresholdEventHandler struct {
	Notification notification.Service
	Logger       *slog.Logger
}

type BalanceThresholdEventHandlerState struct {
	TotalGrants float64 `json:"totalGrants"`
}

var ErrNoBalanceAvailable = errors.New("no balance available")

func (b *BalanceThresholdEventHandler) Handle(ctx context.Context, event snapshot.SnapshotEvent) error {
	if !b.isBalanceThresholdEvent(event) {
		return nil
	}

	// TODO[issue-1364]: this must be cached to prevent going to the DB for each balance.snapshot event
	affectedRulesPaged, err := b.Notification.ListRules(ctx, notification.ListRulesInput{
		Namespaces: []string{event.Namespace.ID},
		Types:      []notification.RuleType{notification.RuleTypeBalanceThreshold},
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

	var errs error
	for _, rule := range affectedRules {
		if !rule.HasEnabledChannels() {
			break
		}

		if err := b.handleRule(ctx, event, rule); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (b *BalanceThresholdEventHandler) handleRule(ctx context.Context, balSnapshot snapshot.SnapshotEvent, rule notification.Rule) error {
	// Check 1: do we have a threshold we should create an event for?

	threshold, err := getHighestMatchingThreshold(rule.Config.BalanceThreshold.Thresholds, *balSnapshot.Value)
	if err != nil {
		return fmt.Errorf("failed to get highest matching threshold: %w", err)
	}

	if threshold == nil {
		// No matching threshold found => nothing to create an event on
		return nil
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
		OrderBy:             notification.EventOrderByCreatedAt,
		Order:               sortx.OrderDesc,
	})
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}

	createEventInput := createBalanceThresholdEventInput{
		Snapshot:   balSnapshot,
		DedupeHash: periodDedupeHash,
		Threshold:  *threshold,
		RuleID:     rule.ID,
	}

	if len(lastEvents.Items) == 0 {
		// we need to trigger the event, as we have hit a threshold, and have no previous event
		return b.createEvent(ctx, createEventInput)
	}

	lastEvent := lastEvents.Items[0]

	if lastEvent.Payload.Type != notification.EventTypeBalanceThreshold {
		// This should never happen, but let's log it and trigger the event, so that we have a better reference point
		// in place
		b.Logger.Error("last event is not a balance threshold event", slog.String("event_id", lastEvent.ID))
		return b.createEvent(ctx, createEventInput)
	}

	lastEventActualValue, err := getBalanceThreshold(
		lastEvent.Payload.BalanceThreshold.Threshold,
		lastEvent.Payload.BalanceThreshold.Value)
	if err != nil {
		if err == ErrNoBalanceAvailable {
			// In case there are no grants, percentage all percentage rules would match, so let's instead
			// wait until we have some credits to calculate the actual value
			b.Logger.Warn("no balance available skipping event creation", "last_event_id", lastEvent.ID)
			return nil
		}
		return fmt.Errorf("failed to calculate actual value from last event: %w", err)
	}

	if lastEventActualValue.BalanceThreshold != *threshold {
		// The last event was triggered by a different threshold, so we need to trigger a new event
		return b.createEvent(ctx, createEventInput)
	}

	return nil
}

type createBalanceThresholdEventInput struct {
	Snapshot   snapshot.SnapshotEvent
	DedupeHash string
	Threshold  notification.BalanceThreshold
	RuleID     string
}

func (b *BalanceThresholdEventHandler) createEvent(ctx context.Context, in createBalanceThresholdEventInput) error {
	entitlementAPIEntity, err := entitlementdriver.Parser.ToMetered(&in.Snapshot.Entitlement)
	if err != nil {
		return fmt.Errorf("failed to map entitlement value to API: %w", err)
	}

	annotations := notification.Annotations{
		notification.AnnotationEventSubjectKey: in.Snapshot.Subject.Key,
		notification.AnnotationEventFeatureKey: in.Snapshot.Feature.Key,
		notification.AnnotationEventDedupeHash: in.DedupeHash,
	}

	if in.Snapshot.Subject.Id != nil && *in.Snapshot.Subject.Id != "" {
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
			BalanceThreshold: notification.BalanceThresholdPayload{
				Entitlement: *entitlementAPIEntity,
				Feature:     productcatalogdriver.MapFeatureToResponse(in.Snapshot.Feature),
				Subject:     in.Snapshot.Subject.ToAPIModel(),
				Value:       (api.EntitlementValue)(*in.Snapshot.Value),
				Threshold:   in.Threshold,
			},
		},
		RuleID:                   in.RuleID,
		HandlerDeduplicationHash: in.DedupeHash,
	})

	return err
}

func (b *BalanceThresholdEventHandler) isBalanceThresholdEvent(event snapshot.SnapshotEvent) bool {
	if event.Entitlement.EntitlementType != entitlement.EntitlementTypeMetered {
		return false
	}

	// We don't care about delete events
	if event.Operation != snapshot.ValueOperationUpdate {
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
func (b *BalanceThresholdEventHandler) getPeriodsDeduplicationHash(snapshot snapshot.SnapshotEvent, ruleID string) string {
	// Note: this should not happen, but let's be safe here
	currentUsagePeriod := defaultx.WithDefault(
		snapshot.Entitlement.CurrentUsagePeriod, recurrence.Period{
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
		defaultx.WithDefault(snapshot.Entitlement.MeasureUsageFrom, time.Time{}).UTC().Format(time.RFC3339),
	}, "/")

	h := sha256.New()

	h.Write([]byte(source))

	bs := h.Sum(nil)

	// bsnap == balance.snapshot
	// v1 == version 1 (in case we need to change the hashing strategy)
	return fmt.Sprintf("bsnap_v1_%x", bs)
}

type balanceThreshold struct {
	notification.BalanceThreshold

	// NumericThreshold always contains the credit value of the threshold regardless if it's a percentage
	// or a number threshold
	NumericThreshold float64
}

func getTotalGrantsFromValue(value api.EntitlementValue) float64 {
	return *value.Balance + *value.Usage - defaultx.WithDefault(value.Overage, 0)
}

func getBalanceThreshold(threshold notification.BalanceThreshold, eValue api.EntitlementValue) (balanceThreshold, error) {
	switch threshold.Type {
	case api.NUMBER:
		return balanceThreshold{
			BalanceThreshold: threshold,
			NumericThreshold: threshold.Value,
		}, nil
	case api.PERCENT:
		totalGrants := getTotalGrantsFromValue(eValue)

		// In case there are no grants yet, we can't calculate the actual value, we are filtering out the
		// thresholds to prevent event triggering in the following scenario:
		//
		// - A new entitlement is created (and there are balance threshold rules active)
		// - Then the granting is done as a separate step
		//
		// As this would mean that we would trigger a notification for the first activity for 100%
		if totalGrants == 0 {
			return balanceThreshold{}, ErrNoBalanceAvailable
		}

		return balanceThreshold{
			BalanceThreshold: threshold,
			NumericThreshold: totalGrants * threshold.Value / 100,
		}, nil

	default:
		return balanceThreshold{}, errors.New("unknown threshold type")
	}
}

func getHighestMatchingThreshold(thresholds []notification.BalanceThreshold, eValue snapshot.EntitlementValue) (*notification.BalanceThreshold, error) {
	// Let's normalize the thresholds in a single slice with percentages already calculated
	actualValues := make([]balanceThreshold, 0, len(thresholds))

	for _, threshold := range thresholds {
		actualValue, err := getBalanceThreshold(threshold, api.EntitlementValue(eValue))
		if err != nil {
			if err == ErrNoBalanceAvailable {
				continue
			}

			return nil, err
		}

		actualValues = append(actualValues, actualValue)
	}

	// Now we have the actual values, let's sort by the thresholds ensuring that we have stable storing between percentages
	// and numbers

	slices.SortFunc(actualValues, func(b1, b2 balanceThreshold) int {
		result := cmp.Compare(b1.NumericThreshold, b2.NumericThreshold)
		if result != 0 {
			return result
		}

		// If the actual values are the same, let's sort by the underlying representation (percentage ends up being the "bigger" one)
		return cmp.Compare(b1.Type, b2.Type)
	})

	var highest *notification.BalanceThreshold
	for _, threshold := range actualValues {
		if threshold.NumericThreshold > *eValue.Usage {
			break
		}

		highest = &threshold.BalanceThreshold
	}

	return highest, nil
}
