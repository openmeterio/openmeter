package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type TestEventGenerator struct {
	billingService billing.Service
}

func NewTestEventGenerator(billingService billing.Service) *TestEventGenerator {
	return &TestEventGenerator{
		billingService: billingService,
	}
}

type EventGeneratorInput struct {
	Namespace string

	EventType notification.EventType
}

func (i EventGeneratorInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.EventType == "" {
		return errors.New("event type is required")
	}

	return nil
}

func (t *TestEventGenerator) Generate(ctx context.Context, in EventGeneratorInput) (notification.EventPayload, error) {
	if err := in.Validate(); err != nil {
		return notification.EventPayload{}, err
	}

	switch in.EventType {
	case notification.EventTypeBalanceThreshold:
		return t.newTestBalanceThresholdPayload(), nil
	case notification.EventTypeEntitlementReset:
		return t.newTestEntitlementResetPayload(), nil
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		return t.newTestInvoicePayload(ctx, in.Namespace, in.EventType)
	default:
		return notification.EventPayload{}, fmt.Errorf("unsupported event type: %s", in.EventType)
	}
}

func (t *TestEventGenerator) newTestBalanceThresholdPayload() notification.EventPayload {
	payload := t.newTestEntitlementResetPayload()
	payload.Type = notification.EventTypeBalanceThreshold
	payload.BalanceThreshold = &notification.BalanceThresholdPayload{
		EntitlementValuePayloadBase: notification.EntitlementValuePayloadBase(*payload.EntitlementReset),
		Threshold: api.NotificationRuleBalanceThresholdValue{
			Type:  api.NotificationRuleBalanceThresholdValueTypePercent,
			Value: 50,
		},
	}
	return payload
}

func (t *TestEventGenerator) newTestEntitlementResetPayload() notification.EventPayload {
	var (
		now       = time.Now()
		createdAt = now.Add(-24 * time.Hour)
		updatedAt = now.Add(-12 * time.Hour)
		from      = now.Add(-24 * time.Hour)
		to        = now.Add(-12 * time.Hour)
	)

	day := &api.RecurringPeriodInterval{}

	_ = day.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumDAY)

	return notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: notification.EventTypeBalanceThreshold,
		},
		EntitlementReset: &notification.EntitlementResetPayload{
			Entitlement: api.EntitlementMetered{
				CreatedAt: createdAt,
				CurrentUsagePeriod: api.Period{
					From: from,
					To:   to,
				},
				DeletedAt:               nil,
				FeatureId:               "01J5AVN2T6S0RDGJHVNN0BW3F5",
				FeatureKey:              "test-feature-1",
				Id:                      "01J5AVNM7H1PT65RDFWGXXPT47",
				IsSoftLimit:             lo.ToPtr(false),
				IsUnlimited:             lo.ToPtr(true),
				IssueAfterReset:         lo.ToPtr(10.0),
				IssueAfterResetPriority: lo.ToPtr[uint8](5),
				LastReset:               time.Time{},
				MeasureUsageFrom:        time.Time{},
				Metadata: &map[string]string{
					"test-metadata-key": "test-metadata-value",
				},
				PreserveOverageAtReset: lo.ToPtr(true),
				SubjectKey:             "test-subject-1",
				Type:                   api.EntitlementMeteredTypeMetered,
				UpdatedAt:              updatedAt,
				UsagePeriod: api.RecurringPeriod{
					Anchor:      from,
					Interval:    *day,
					IntervalISO: "P1D",
				},
			},
			Feature: api.Feature{
				ArchivedAt: nil,
				CreatedAt:  createdAt,
				DeletedAt:  nil,
				Id:         "01J5AVN2T6S0RDGJHVNN0BW3F5",
				Key:        "test-feature-1",
				Metadata: &map[string]string{
					"test-metadata-key": "test-metadata-value",
				},
				MeterGroupByFilters: nil,
				MeterSlug:           lo.ToPtr("test-meter-1"),
				Name:                "test-meter-1",
				UpdatedAt:           updatedAt,
			},
			Subject: api.Subject{
				CurrentPeriodEnd:   lo.ToPtr(from),
				CurrentPeriodStart: lo.ToPtr(to),
				DisplayName:        lo.ToPtr("Test Subject 1"),
				Id:                 "01J5AW0ZD6T8624PCK0Q5TYX71",
				Key:                "test-subject-1",
				Metadata: &map[string]interface{}{
					"test-metadata-key": "test-metadata-value",
				},
				StripeCustomerId: lo.ToPtr("01J5AW2XS6DYHH7E9PNJSQJ341"),
			},

			Value: api.EntitlementValue{
				Balance:   lo.ToPtr(10_000.0),
				HasAccess: true,
				Overage:   lo.ToPtr(99.0),
				Usage:     lo.ToPtr(5_001.0),
			},
		},
	}
}

func (t *TestEventGenerator) newTestInvoicePayload(ctx context.Context, namespace string, eventType notification.EventType) (notification.EventPayload, error) {
	now := time.Now().Truncate(time.Second).In(time.UTC)

	invoice, err := t.billingService.SimulateInvoice(ctx, billing.SimulateInvoiceInput{
		Namespace: namespace,
		Customer: &customer.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: namespace,
				ID:        ulid.Make().String(),
				Name:      "Test Customer",
				CreatedAt: now,
				UpdatedAt: now,
			}),
			Key: lo.ToPtr("test-customer-1"),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test-subject-1"},
			},
			PrimaryEmail: lo.ToPtr("test-customer-1@example.com"),
			Currency:     lo.ToPtr(currencyx.Code(currency.USD)),
		},

		Number:   lo.ToPtr("TEST-INV-1"),
		Currency: currencyx.Code(currency.USD),
		Lines: billing.NewInvoiceLines([]*billing.Line{
			billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
				Namespace: namespace,
				ID:        ulid.Make().String(),
				CreatedAt: now,
				UpdatedAt: now,

				ManagedBy: billing.ManuallyManagedLine,

				Name: "test flat fee",
				Period: billing.Period{
					Start: now.Add(-time.Hour * 24 * 30),
					End:   now,
				},
				InvoiceAt: now,

				PerUnitAmount: alpacadecimal.NewFromInt(1000),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		}),
	})
	if err != nil {
		return notification.EventPayload{}, err
	}

	eventInvoice, err := billing.NewEventInvoice(invoice)
	if err != nil {
		return notification.EventPayload{}, err
	}

	return notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: eventType,
		},
		Invoice: &eventInvoice,
	}, nil
}
