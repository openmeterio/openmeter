package internal

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/rickb777/period"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewTestEventPayload(eventType notification.EventType) notification.EventPayload {
	switch eventType {
	case notification.EventTypeBalanceThreshold:
		return newTestBalanceThresholdPayload()
	case notification.EventTypeEntitlementReset:
		return newTestEntitlementResetPayload()
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		return newTestInvoicePayload(eventType)
	default:
		return notification.EventPayload{}
	}
}

func newTestBalanceThresholdPayload() notification.EventPayload {
	payload := newTestEntitlementResetPayload()
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

func newTestEntitlementResetPayload() notification.EventPayload {
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

func newTestInvoicePayload(eventType notification.EventType) notification.EventPayload {
	now := time.Now().Truncate(time.Microsecond).In(time.UTC)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)
	createdAt := now.Add(time.Hour)
	updatedAt := now.Add(time.Hour)
	dueAt := now.Add(24 * time.Hour)
	draftUntil := now.Add(4 * time.Hour)
	collectionAt := now.Add(5 * time.Hour)

	return notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: eventType,
		},
		Invoice: &notification.InvoicePayload{
			Invoice: billing.Invoice{
				InvoiceBase: billing.InvoiceBase{
					Namespace:   "test",
					ID:          "01JT3M7RES4JEWD82223RZCVX5",
					Number:      "DRAFT-TEST-USD-1",
					Description: lo.ToPtr("Test invoice event"),
					Type:        billing.InvoiceTypeStandard,
					Metadata: map[string]string{
						"test-metadata-key": "test-metadata-value",
					},
					Currency: currencyx.Code(currency.USD),
					Status:   billing.InvoiceStatusDraftCreated,
					StatusDetails: billing.InvoiceStatusDetails{
						Immutable: true,
						Failed:    false,
						AvailableActions: billing.InvoiceAvailableActions{
							Advance: &billing.InvoiceAvailableActionDetails{
								ResultingState: "",
							},
							Approve: &billing.InvoiceAvailableActionDetails{
								ResultingState: "",
							},
							Delete: &billing.InvoiceAvailableActionDetails{
								ResultingState: "",
							},
							Retry: &billing.InvoiceAvailableActionDetails{
								ResultingState: "",
							},
							Void: &billing.InvoiceAvailableActionDetails{
								ResultingState: "",
							},
							Invoice: &billing.InvoiceAvailableActionInvoiceDetails{},
						},
					},
					Period: &billing.Period{
						Start: periodStart,
						End:   periodEnd,
					},
					DueAt:            &dueAt,
					CreatedAt:        createdAt,
					UpdatedAt:        updatedAt,
					VoidedAt:         nil,
					DraftUntil:       &draftUntil,
					IssuedAt:         &issueAt,
					DeletedAt:        nil,
					SentToCustomerAt: nil,
					CollectionAt:     &collectionAt,
					Customer: billing.InvoiceCustomer{
						CustomerID: "01JT3M76AYWTWZN91KFNJ85G21",
						Name:       "Test Customer",
						BillingAddress: &models.Address{
							Country:     lo.ToPtr(models.CountryCode("US")),
							PostalCode:  lo.ToPtr("12345"),
							State:       lo.ToPtr("NY"),
							City:        lo.ToPtr("New York"),
							Line1:       lo.ToPtr("1234 Test St"),
							Line2:       lo.ToPtr("Apt 1"),
							PhoneNumber: lo.ToPtr("1234567890"),
						},
						UsageAttribution: customer.CustomerUsageAttribution{
							SubjectKeys: []string{"test-subject"},
						},
					},
					Supplier: billing.SupplierContact{
						Name: "Awesome Supplier",
						Address: models.Address{
							Country: lo.ToPtr(models.CountryCode("US")),
						},
					},
					Workflow: billing.InvoiceWorkflow{
						AppReferences: billing.ProfileAppReferences{
							Tax: billing.AppReference{
								ID:   "",
								Type: "",
							},
							Invoicing: billing.AppReference{
								ID:   "",
								Type: "",
							},
							Payment: billing.AppReference{
								ID:   "",
								Type: "",
							},
						},
						Apps: &billing.ProfileApps{
							Tax:       nil,
							Invoicing: nil,
							Payment:   nil,
						},
						SourceBillingProfileID: "",
						Config: billing.WorkflowConfig{
							Collection: billing.CollectionConfig{
								Alignment: "",
								Interval: isodate.Period{
									Period: period.Period{},
								},
							},
							Invoicing: billing.InvoicingConfig{
								AutoAdvance: false,
								DraftPeriod: isodate.Period{
									Period: period.Period{},
								},
								DueAfter: isodate.Period{
									Period: period.Period{},
								},
								ProgressiveBilling: false,
								DefaultTaxConfig: &productcatalog.TaxConfig{
									Behavior: nil,
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "",
									},
								},
							},
							Payment: billing.PaymentConfig{
								CollectionMethod: "",
							},
						},
					},
					ExternalIDs: billing.InvoiceExternalIDs{
						Invoicing: "",
						Payment:   "",
					},
				},
				Lines: billing.LineChildren{
					Option: mo.Option[[]*billing.Line]{},
				},
				ValidationIssues: nil,
				Totals: billing.Totals{
					Amount:              alpacadecimal.NewFromInt(1000),
					ChargesTotal:        alpacadecimal.NewFromInt(900),
					DiscountsTotal:      alpacadecimal.NewFromInt(100),
					TaxesInclusiveTotal: alpacadecimal.NewFromInt(50),
					TaxesExclusiveTotal: alpacadecimal.NewFromInt(50),
					TaxesTotal:          alpacadecimal.NewFromInt(100),
					Total:               alpacadecimal.NewFromInt(1050),
				},
				ExpandedFields: billing.InvoiceExpand{
					Preceding:                   true,
					Lines:                       true,
					DeletedLines:                true,
					SplitLines:                  true,
					RecalculateGatheringInvoice: true,
				},
			},
			Apps: billing.InvoiceApps{
				Tax: app.EventApp{
					AppBase: app.AppBase{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "",
							},
							ManagedModel: models.ManagedModel{
								CreatedAt: time.Time{},
								UpdatedAt: time.Time{},
								DeletedAt: &time.Time{},
							},
							ID:          "",
							Description: nil,
							Name:        "",
						},
						Type:    "",
						Status:  "",
						Default: false,
						Listing: app.MarketplaceListing{
							Type:           "",
							Name:           "",
							Description:    "",
							Capabilities:   nil,
							InstallMethods: nil,
						},
						Metadata: nil,
					},
					AppData: nil,
				},
				Payment: app.EventApp{
					AppBase: app.AppBase{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "",
							},
							ManagedModel: models.ManagedModel{
								CreatedAt: time.Time{},
								UpdatedAt: time.Time{},
								DeletedAt: &time.Time{},
							},
							ID:          "",
							Description: nil,
							Name:        "",
						},
						Type:    "",
						Status:  "",
						Default: false,
						Listing: app.MarketplaceListing{
							Type:           "",
							Name:           "",
							Description:    "",
							Capabilities:   nil,
							InstallMethods: nil,
						},
						Metadata: nil,
					},
					AppData: nil,
				},
				Invoicing: app.EventApp{
					AppBase: app.AppBase{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "",
							},
							ManagedModel: models.ManagedModel{
								CreatedAt: time.Time{},
								UpdatedAt: time.Time{},
								DeletedAt: &time.Time{},
							},
							ID:          "",
							Description: nil,
							Name:        "",
						},
						Type:    "",
						Status:  "",
						Default: false,
						Listing: app.MarketplaceListing{
							Type:           "",
							Name:           "",
							Description:    "",
							Capabilities:   nil,
							InstallMethods: nil,
						},
						Metadata: nil,
					},
					AppData: nil,
				},
			},
		},
	}
}
