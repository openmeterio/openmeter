package notification

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/notification/events"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
	"github.com/openmeterio/openmeter/openmeter/app"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/consumer"
	notificationhttpdriver "github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// LegacyEventV3TestSuite checks that Events produced by the real notification
// consumers can be read through the v3 API: the response must match the v3 OpenAPI
// spec and carry the same values as the v1 API. Fixtures built from the v1 schemas
// cannot catch producer output the schema does not allow.
type LegacyEventV3TestSuite struct {
	Env TestEnv

	namespace string
	feature   feature.Feature
	channel   notification.Channel
	snapshot  consumer.EntitlementSnapshotHandler
	invoice   consumer.InvoiceEventHandler

	v1 http.Handler
	v3 http.Handler
	// v3ResponseErrors collects the response validation failures of the v3 router.
	v3ResponseErrors []error
}

func (s *LegacyEventV3TestSuite) Setup(ctx context.Context, t *testing.T) {
	t.Helper()

	s.namespace = NewTestNamespace(t)
	service := s.Env.Notification()

	err := s.Env.Meter().ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID:              ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String(),
				NamespacedModel: models.NamespacedModel{Namespace: s.namespace},
				ManagedModel:    models.ManagedModel{CreatedAt: time.Now(), UpdatedAt: time.Now()},
				Name:            "Meter 1",
			},
			Key:           TestMeterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "request",
			ValueProperty: lo.ToPtr("$.duration_ms"),
			GroupBy: map[string]string{
				"method": "$.method",
				"path":   "$.path",
			},
		},
	})
	require.NoError(t, err)

	m, err := s.Env.Meter().GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: s.namespace, IDOrSlug: TestMeterSlug})
	require.NoError(t, err)

	s.feature, err = s.Env.Feature().CreateFeature(ctx, feature.CreateFeatureInputs{
		Name: TestFeatureName,
		// Real feature keys are snake_case; the shared TestFeatureKey is not a valid one.
		Key:                 "api_call",
		Namespace:           s.namespace,
		MeterID:             convert.ToPointer(m.ID),
		MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(m.GroupBy),
	})
	require.NoError(t, err)

	channel, err := service.CreateChannel(ctx, NewCreateChannelInput(s.namespace, "LegacyEventV3"))
	require.NoError(t, err)

	s.channel = *channel

	// One rule per legacy event type the consumers below produce.
	_, err = service.CreateRule(ctx, NewCreateRuleInput(s.namespace, "BalanceThreshold", s.channel.ID))
	require.NoError(t, err)

	for _, eventType := range []notification.EventType{notification.EventTypeEntitlementReset, notification.EventTypeInvoiceCreated} {
		in := NewCreateRuleInput(s.namespace, string(eventType), s.channel.ID)
		in.Type = eventType
		in.Config = notification.RuleConfig{RuleConfigMeta: notification.RuleConfigMeta{Type: eventType}}

		switch eventType {
		case notification.EventTypeEntitlementReset:
			in.Config.EntitlementReset = &notification.EntitlementResetRuleConfig{}
		default:
			in.Config.Invoice = &notification.InvoiceRuleConfig{}
		}

		_, err = service.CreateRule(ctx, in)
		require.NoError(t, err)
	}

	s.snapshot = consumer.EntitlementSnapshotHandler{Notification: service, Logger: slog.Default()}
	s.invoice = consumer.InvoiceEventHandler{Notification: service, Logger: slog.Default()}

	s.v1 = s.newV1Router(service)
	s.v3 = s.newV3Router(t, service)
}

func (s *LegacyEventV3TestSuite) newV1Router(service notification.Service) http.Handler {
	handler := notificationhttpdriver.New(namespacedriver.StaticNamespaceDecoder(s.namespace), service, nil)

	router := chi.NewRouter()
	router.Get("/api/v1/notification/events", func(w http.ResponseWriter, r *http.Request) {
		handler.ListEvents().With(notificationhttpdriver.ListEventsParams{}).ServeHTTP(w, r)
	})
	router.Get("/api/v1/notification/events/{eventId}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetEvent().With(chi.URLParam(r, "eventId")).ServeHTTP(w, r)
	})

	return router
}

// newV3Router serves the v3 Event endpoints and validates every response against the
// embedded v3 OpenAPI spec, collecting failures in v3ResponseErrors.
func (s *LegacyEventV3TestSuite) newV3Router(t *testing.T, service notification.Service) http.Handler {
	t.Helper()

	doc, err := api.GetSpec()
	require.NoError(t, err)

	validationRouter, err := oasmiddleware.NewValidationRouter(t.Context(), doc, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	handler := events.New(func(ctx context.Context) (string, error) { return s.namespace, nil }, service)

	router := chi.NewRouter()
	router.Use(oasmiddleware.ValidateResponse(validationRouter, oasmiddleware.ValidateResponseOption{
		ResponseValidationErrorHook: func(err error, _ *http.Request) {
			s.v3ResponseErrors = append(s.v3ResponseErrors, err)
		},
		FilterOptions: &openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			MultiError:         true,
		},
		RouteFilterHook: func(*routers.Route) bool { return true },
	}))
	router.Get("/openmeter/notification/events", func(w http.ResponseWriter, r *http.Request) {
		handler.ListNotificationEvents().With(api.ListBillingNotificationEventsParams{}).ServeHTTP(w, r)
	})
	router.Get("/openmeter/notification/events/{eventId}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetNotificationEvent().With(chi.URLParam(r, "eventId")).ServeHTTP(w, r)
	})

	return router
}

func (s *LegacyEventV3TestSuite) get(t *testing.T, router http.Handler, target string) []byte {
	t.Helper()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	require.Equal(t, http.StatusOK, rec.Code, "GET %s: %s", target, rec.Body.String())

	return rec.Body.Bytes()
}

// readEvent reads the Event through both APIs, requires the v3 responses to be
// schema-valid and returns the v1 and v3 payloads.
func (s *LegacyEventV3TestSuite) readEvent(t *testing.T, eventID string, eventType notification.EventType) (v1Payload, v3Payload map[string]any) {
	t.Helper()

	s.v3ResponseErrors = nil

	v1Get := s.get(t, s.v1, "/api/v1/notification/events/"+eventID)
	v3Get := s.get(t, s.v3, "/openmeter/notification/events/"+eventID)
	v1List := s.get(t, s.v1, "/api/v1/notification/events")
	v3List := s.get(t, s.v3, "/openmeter/notification/events")

	require.Empty(t, s.v3ResponseErrors, "v3 responses must match the v3 OpenAPI spec")

	var v1Event, v3Event map[string]any
	require.NoError(t, json.Unmarshal(v1Get, &v1Event))
	require.NoError(t, json.Unmarshal(v3Get, &v3Event))

	require.Equal(t, eventID, v1Event["id"])
	require.Equal(t, v1Event["id"], v3Event["id"])
	require.Equal(t, string(eventType), v1Event["type"])
	require.Equal(t, v1Event["type"], v3Event["type"])

	// The list must carry the same Event as the get by id on both APIs.
	require.Equal(t, v1Event["payload"], findListedEvent(t, v1List, eventID)["payload"])
	require.Equal(t, v3Event["payload"], findListedEvent(t, v3List, eventID)["payload"])

	require.Equal(t, jsonLeaves(v1Event["payload"]), jsonLeaves(v3Event["payload"]),
		"v3 payload must carry the same values as the v1 payload")

	v1Payload, _ = v1Event["payload"].(map[string]any)
	v3Payload, _ = v3Event["payload"].(map[string]any)
	require.NotNil(t, v1Payload)
	require.NotNil(t, v3Payload)

	return v1Payload, v3Payload
}

func findListedEvent(t *testing.T, listBody []byte, eventID string) map[string]any {
	t.Helper()

	var list struct {
		Data  []map[string]any `json:"data"`
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(listBody, &list))

	for _, item := range slices.Concat(list.Data, list.Items) {
		if item["id"] == eventID {
			return item
		}
	}

	require.Failf(t, "event not listed", "event %s not found in %s", eventID, listBody)

	return nil
}

// jsonLeaves lists every scalar value of a decoded JSON document in a stable order,
// ignoring where it sits, so field renames (camelCase to snake_case) do not matter.
func jsonLeaves(value any) []string {
	var leaves []string

	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for _, item := range v {
				walk(item)
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		default:
			leaves = append(leaves, fmt.Sprintf("%v", v))
		}
	}
	walk(value)

	slices.Sort(leaves)

	return leaves
}

// newestEventID returns the id of the most recently created Event of the type.
func (s *LegacyEventV3TestSuite) newestEventID(ctx context.Context, t *testing.T, eventType notification.EventType) string {
	t.Helper()

	resp, err := s.Env.Notification().ListEvents(ctx, notification.ListEventsInput{
		Namespaces: []string{s.namespace},
		OrderBy:    notification.OrderByCreatedAt,
		Order:      sortx.OrderDesc,
	})
	require.NoError(t, err)

	for _, event := range resp.Items {
		if event.Type == eventType {
			return event.ID
		}
	}

	require.Failf(t, "no event", "no %s event created", eventType)

	return ""
}

type balanceThresholdScenario struct {
	name     string
	customer customer.Customer
	subject  subject.Subject
}

func (s *LegacyEventV3TestSuite) TestBalanceThresholdAndReset(ctx context.Context, t *testing.T) {
	scenarios := []balanceThresholdScenario{
		{
			// A customer attributed to a stored subject.
			name: "stored subject",
			customer: customer.Customer{
				ManagedResource:  models.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: s.namespace}, ID: NewTestULID(t), Name: "Stored"},
				Key:              lo.ToPtr("stored-customer"),
				UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"stored-subject"}},
			},
			subject: subject.Subject{Namespace: s.namespace, Id: NewTestULID(t), Key: "stored-subject"},
		},
		{
			// The balance worker falls back to a subject without id when the subject row is missing.
			name: "subject row missing",
			customer: customer.Customer{
				ManagedResource:  models.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: s.namespace}, ID: NewTestULID(t), Name: "Missing"},
				Key:              lo.ToPtr("missing-customer"),
				UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"missing-subject"}},
			},
			subject: subject.Subject{Namespace: s.namespace, Key: "missing-subject"},
		},
		{
			// A customer without usage attribution has no subject at all.
			name: "no usage attribution",
			customer: customer.Customer{
				ManagedResource: models.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: s.namespace}, ID: NewTestULID(t), Name: "Unattributed"},
				Key:             lo.ToPtr("unattributed-customer"),
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// given an entitlement of the customer that crosses the 95% usage threshold
			snapshotEvent := NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
				Feature: s.feature,
				Value: snapshot.EntitlementValue{
					Balance:                   convert.ToPointer(4.0),
					Usage:                     convert.ToPointer(96.0),
					TotalAvailableGrantAmount: convert.ToPointer(100.0),
				},
				Namespace: s.namespace,
			})
			snapshotEvent.Entitlement.ID = NewTestULID(t)
			snapshotEvent.Entitlement.CustomerID = sc.customer.ID
			snapshotEvent.Customer = sc.customer
			snapshotEvent.Subject = sc.subject

			// when the real consumer handles the threshold crossing and a reset
			require.NoError(t, s.snapshot.Handle(ctx, snapshotEvent))

			// The reset handler skips the Event when anything was created in the current
			// usage period, so the reset snapshot reports a period nothing is created in.
			resetEvent := snapshotEvent
			resetEvent.Operation = snapshot.ValueOperationReset
			resetEvent.Entitlement.CurrentUsagePeriod = &timeutil.ClosedPeriod{
				From: time.Now().Add(48 * time.Hour),
				To:   time.Now().Add(72 * time.Hour),
			}
			require.NoError(t, s.snapshot.Handle(ctx, resetEvent))

			// then both APIs return the same content and v3 matches the spec
			for _, eventType := range []notification.EventType{notification.EventTypeBalanceThreshold, notification.EventTypeEntitlementReset} {
				t.Run(string(eventType), func(t *testing.T) {
					_, v3Payload := s.readEvent(t, s.newestEventID(ctx, t, eventType), eventType)

					data, ok := v3Payload["data"].(map[string]any)
					require.True(t, ok, "payload.data must be an object: %v", v3Payload)

					v3Subject, ok := data["subject"].(map[string]any)
					require.True(t, ok, "payload.data.subject must be an object: %v", data)
					require.Equal(t, sc.subject.Id, v3Subject["id"])
					require.Equal(t, sc.subject.Key, v3Subject["key"])

					v3Entitlement, ok := data["entitlement"].(map[string]any)
					require.True(t, ok, "payload.data.entitlement must be an object: %v", data)

					// Both are empty for a customer without usage attribution.
					require.Equal(t, sc.subject.Key, v3Entitlement["subject_key"])
				})
			}
		})
	}
}

func (s *LegacyEventV3TestSuite) TestInvoiceCreated(ctx context.Context, t *testing.T) {
	scenarios := []struct {
		name             string
		usageAttribution *customer.CustomerUsageAttribution
	}{
		{name: "attributed customer", usageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"invoiced-subject"}}},
		// A customer without subject keys must have a key; its invoice gets empty subject keys.
		{name: "key only"},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// given a draft invoice of the customer
			invoice := billing.StandardInvoice{
				StandardInvoiceBase: billing.StandardInvoiceBase{
					Namespace: s.namespace,
					ID:        NewTestULID(t),
					Number:    "INV-0001",
					Type:      billing.InvoiceTypeStandard,
					Currency:  currencyx.FiatCode("USD"),
					Status:    billing.StandardInvoiceStatusDraftCreated,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
					Customer: billing.NewInvoiceCustomer(customer.Customer{
						ManagedResource:  models.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: s.namespace}, ID: NewTestULID(t), Name: "Invoiced"},
						Key:              lo.ToPtr("invoiced-customer"),
						UsageAttribution: sc.usageAttribution,
					}),
					Supplier: billing.SupplierContact{
						Name:    "Supplier",
						Address: models.Address{Country: lo.ToPtr(models.CountryCode("US"))},
					},
					Workflow: billing.InvoiceWorkflow{
						SourceBillingProfileID: NewTestULID(t),
						Config: billing.WorkflowConfig{
							Collection: billing.CollectionConfig{
								Alignment: billing.AlignmentKindSubscription,
								Interval:  datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0),
							},
							Invoicing: billing.InvoicingConfig{
								DraftPeriod:                  datetime.NewISODuration(0, 0, 0, 1, 0, 0, 0),
								DueAfter:                     datetime.NewISODuration(0, 0, 0, 30, 0, 0, 0),
								SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillFullPeriod,
							},
							Payment: billing.PaymentConfig{CollectionMethod: billing.CollectionMethodChargeAutomatically},
						},
					},
				},
			}

			sandboxApp := app.EventApp{
				AppBase: app.AppBase{
					ManagedResource: models.ManagedResource{
						NamespacedModel: models.NamespacedModel{Namespace: s.namespace},
						ID:              NewTestULID(t),
						ManagedModel:    models.ManagedModel{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
						Name:            "Sandbox",
					},
					Type:    app.AppTypeSandbox,
					Status:  app.AppStatusReady,
					Listing: appsandbox.MarketplaceListing,
				},
			}

			// when the real invoice consumer handles the event
			require.NoError(t, s.invoice.Handle(ctx, billing.EventStandardInvoice{
				Invoice: invoice,
				Apps:    billing.InvoiceApps{Tax: sandboxApp, Payment: sandboxApp, Invoicing: sandboxApp},
			}, notification.EventTypeInvoiceCreated))

			// then both APIs return the same content and v3 matches the spec
			s.readEvent(t, s.newestEventID(ctx, t, notification.EventTypeInvoiceCreated), notification.EventTypeInvoiceCreated)
		})
	}
}
