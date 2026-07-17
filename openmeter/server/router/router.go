package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/openmeterio/openmeter/api"
	v3apierrors "github.com/openmeterio/openmeter/api/v3/apierrors"
	currencyhandler "github.com/openmeterio/openmeter/api/v3/handlers/currencies"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appcustominvoicinghttpdriver "github.com/openmeterio/openmeter/openmeter/app/custominvoicing/httpdriver"
	apphttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/credit"
	creditdriver "github.com/openmeterio/openmeter/openmeter/credit/driver"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/debug"
	debug_httpdriver "github.com/openmeterio/openmeter/openmeter/debug/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	entitlementdriverv2 "github.com/openmeterio/openmeter/openmeter/entitlement/driver/v2"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/governance"
	infohttpdriver "github.com/openmeterio/openmeter/openmeter/info/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	ingesthttpdriver "github.com/openmeterio/openmeter/openmeter/ingest/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meterhttphandler "github.com/openmeterio/openmeter/openmeter/meter/httphandler"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	metereventhttphandler "github.com/openmeterio/openmeter/openmeter/meterevent/httphandler"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationhttpdriver "github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/portal"
	portalhttphandler "github.com/openmeterio/openmeter/openmeter/portal/httphandler"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	addonhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/httpdriver"
	productcatalog_httpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/driver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	planaddonhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/httpdriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	progresshttpdriver "github.com/openmeterio/openmeter/openmeter/progressmanager/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjecthttphandler "github.com/openmeterio/openmeter/openmeter/subject/httphandler"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionaddonhttpdriver "github.com/openmeterio/openmeter/openmeter/subscription/addon/http"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

func init() {
	// See https://github.com/getkin/kin-openapi/issues/640
	openapi3filter.RegisterBodyDecoder("application/cloudevents+json", jsonBodyDecoder)
	openapi3filter.RegisterBodyDecoder("application/cloudevents-batch+json", jsonBodyDecoder)
}

func jsonBodyDecoder(body io.Reader, header http.Header, schema *openapi3.SchemaRef, encFn openapi3filter.EncodingFn) (interface{}, error) {
	var value interface{}
	if err := json.NewDecoder(body).Decode(&value); err != nil {
		return nil, &openapi3filter.ParseError{Kind: openapi3filter.KindInvalidFormat, Cause: err}
	}
	return value, nil
}

type IngestHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, namespace string)
}

type Config struct {
	NamespaceDecoder            namespacedriver.NamespaceDecoder
	Addon                       addon.Service
	App                         app.Service
	AppStripe                   appstripe.Service
	AppCustomInvoicing          appcustominvoicing.SyncService
	Billing                     billing.Service
	BillingFeatureSwitches      config.BillingFeatureSwitchesConfiguration
	ChargeService               billingcharges.ChargeService
	Credits                     config.CreditsConfiguration
	UnitConfig                  config.UnitConfigConfiguration
	CurrencyService             currencies.Service
	CostService                 cost.Service
	CreditGrantService          creditgrant.Service
	Ledger                      ledger.Ledger
	AccountResolver             ledger.AccountResolver
	Customer                    customer.Service
	CustomerBalanceFacade       *customerbalance.Facade
	DebugConnector              debug.DebugConnector
	EntitlementConnector        entitlement.Service
	EntitlementBalanceConnector meteredentitlement.Connector
	ErrorHandler                errorsx.Handler
	FeatureConnector            feature.FeatureConnector
	GovernanceService           governance.Service
	GrantConnector              credit.GrantConnector
	GrantRepo                   grant.Repo
	IngestService               ingest.Service
	LLMCostService              llmcost.Service
	Logger                      *slog.Logger
	MeterManageService          meter.ManageService
	MeterEventService           meterevent.Service
	Notification                notification.Service
	Plan                        plan.Service
	PlanAddon                   planaddon.Service
	PlanSubscriptionService     plansubscription.PlanSubscriptionService
	PortalCORSEnabled           bool
	Portal                      portal.Service
	ProgressManager             progressmanager.Service
	StreamingConnector          streaming.Connector
	SubscriptionService         subscription.Service
	SubscriptionAddonService    subscriptionaddon.Service
	SubscriptionWorkflowService subscriptionworkflow.Service
	SubjectService              subject.Service
	TaxCodeService              taxcode.Service
	FeatureGate                 *featuregate.FeatureGateChecker
}

func (c Config) Validate() error {
	if c.NamespaceDecoder == nil {
		return errors.New("namespace decoder is required")
	}

	if c.ErrorHandler == nil {
		return errors.New("error handler is required")
	}

	if c.IngestService == nil {
		return errors.New("ingest service is required")
	}

	// Validate connectors
	if c.Addon == nil {
		return errors.New("add-on service is required")
	}

	if c.App == nil {
		return errors.New("app service is required")
	}

	if c.AppStripe == nil {
		return errors.New("app stripe service is required")
	}

	if c.AppCustomInvoicing == nil {
		return errors.New("app custom invoicing service is required")
	}

	if c.ChargeService == nil {
		return errors.New("charge service is required")
	}

	if c.Customer == nil {
		return errors.New("customer service is required")
	}

	if c.CurrencyService == nil {
		return errors.New("currency service is required")
	}

	if c.Billing == nil {
		return errors.New("billing service is required")
	}

	if c.DebugConnector == nil {
		return errors.New("debug connector is required")
	}

	if c.FeatureConnector == nil {
		return errors.New("feature connector is required")
	}

	if c.EntitlementConnector == nil {
		return errors.New("entitlement connector is required")
	}

	if c.EntitlementBalanceConnector == nil {
		return errors.New("entitlement balance connector is required")
	}

	if c.GrantConnector == nil {
		return errors.New("grant connector is required")
	}

	if c.GrantRepo == nil {
		return errors.New("grant repo is required")
	}

	if c.LLMCostService == nil {
		return errors.New("llm cost service is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.MeterManageService == nil {
		return errors.New("meter manage service is required")
	}

	if c.MeterEventService == nil {
		return errors.New("meter event service is required")
	}

	if c.Notification == nil {
		return errors.New("notification service is required")
	}

	if c.Plan == nil {
		return errors.New("plan service is required")
	}

	if c.PlanAddon == nil {
		return errors.New("plan add-on service is required")
	}

	if c.PlanSubscriptionService == nil {
		return errors.New("plan subscription service is required")
	}

	if c.Portal == nil {
		return errors.New("portal service is required")
	}

	if c.ProgressManager == nil {
		return errors.New("progress manager service is required")
	}

	if c.StreamingConnector == nil {
		return errors.New("streaming connector is required")
	}

	if c.SubscriptionWorkflowService == nil {
		return errors.New("subscription workflow service is required")
	}

	if c.SubscriptionService == nil {
		return errors.New("subscription service is required")
	}

	if c.SubscriptionAddonService == nil {
		return errors.New("subscription addon service is required")
	}

	if c.SubjectService == nil {
		return errors.New("subject service is required")
	}

	if c.TaxCodeService == nil {
		return errors.New("tax code service is required")
	}

	if c.FeatureGate == nil {
		return errors.New("feature gate is required")
	}

	if err := c.FeatureGate.Validate(); err != nil {
		return err
	}

	if c.GovernanceService == nil {
		return errors.New("governance service is required")
	}

	return nil
}

type Router struct {
	config Config

	addonHandler              addonhttpdriver.AddonHandler
	appHandler                apphttpdriver.Handler
	appStripeHandler          appstripehttpdriver.AppStripeHandler
	appCustomInvoicingHandler appcustominvoicinghttpdriver.Handler
	billingHandler            billinghttpdriver.Handler
	currencyHandler           currencyhandler.Handler
	featureHandler            productcatalog_httpdriver.FeatureHandler
	planHandler               planhttpdriver.Handler
	planAddonHandler          planaddonhttpdriver.Handler
	subscriptionHandler       subscriptionhttpdriver.Handler
	subscriptionAddonHandler  subscriptionaddonhttpdriver.Handler
	creditHandler             creditdriver.GrantHandler
	debugHandler              debug_httpdriver.DebugHandler
	customerHandler           customerhttpdriver.CustomerHandler
	entitlementHandler        entitlementdriver.EntitlementHandler
	entitlementV2Handler      entitlementdriverv2.EntitlementHandler
	meterHandler              meterhttphandler.Handler
	ingestHandler             ingesthttpdriver.Handler
	meterEventHandler         metereventhttphandler.Handler
	meteredEntitlementHandler entitlementdriver.MeteredEntitlementHandler
	portalHandler             portalhttphandler.Handler
	notificationHandler       notificationhttpdriver.Handler
	progressHandler           progresshttpdriver.Handler
	infoHandler               infohttpdriver.Handler
	subjectHandler            subjecthttphandler.Handler
}

// Make sure we conform to ServerInterface
var _ api.ServerInterface = (*Router)(nil)

func NewRouter(config Config) (*Router, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid router config: %w", err)
	}

	router := &Router{
		config: config,
	}

	if err := router.initHandlers(config); err != nil {
		return nil, err
	}

	return router, nil
}

func (router *Router) initHandlers(config Config) error {
	var err error

	router.debugHandler, err = debug_httpdriver.NewDebugHandler(
		config.NamespaceDecoder,
		config.DebugConnector,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("debug handler: %w", err)
	}

	router.featureHandler, err = productcatalog_httpdriver.NewFeatureHandler(
		config.FeatureConnector,
		config.NamespaceDecoder,
		config.MeterManageService,
		config.LLMCostService,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("feature handler: %w", err)
	}

	router.entitlementHandler, err = entitlementdriver.NewEntitlementHandler(
		config.EntitlementConnector,
		config.Customer,
		config.SubjectService,
		config.NamespaceDecoder,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("entitlement handler: %w", err)
	}

	// V2 entitlement handler for customer-scoped operations
	router.entitlementV2Handler, err = entitlementdriverv2.NewEntitlementHandler(
		config.EntitlementConnector,
		config.EntitlementBalanceConnector,
		config.Customer,
		config.NamespaceDecoder,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("entitlement v2 handler: %w", err)
	}

	router.meterHandler, err = meterhttphandler.New(
		config.NamespaceDecoder,
		config.Customer,
		config.MeterManageService,
		config.StreamingConnector,
		config.SubjectService,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("meter handler: %w", err)
	}

	router.ingestHandler, err = ingesthttpdriver.New(
		config.NamespaceDecoder,
		config.IngestService,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("ingest handler: %w", err)
	}

	router.meterEventHandler, err = metereventhttphandler.New(
		config.NamespaceDecoder,
		config.MeterEventService,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("meter event handler: %w", err)
	}

	router.meteredEntitlementHandler, err = entitlementdriver.NewMeteredEntitlementHandler(
		config.EntitlementConnector,
		config.EntitlementBalanceConnector,
		config.Customer,
		config.SubjectService,
		config.NamespaceDecoder,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("metered entitlement handler: %w", err)
	}

	router.creditHandler, err = creditdriver.NewGrantHandler(
		config.NamespaceDecoder,
		config.GrantConnector,
		config.GrantRepo,
		config.Customer,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("grant handler: %w", err)
	}

	router.notificationHandler, err = notificationhttpdriver.New(
		config.NamespaceDecoder,
		config.Notification,
		config.Billing,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("notification handler: %w", err)
	}

	router.infoHandler = infohttpdriver.New(
		httptransport.WithErrorHandler(config.ErrorHandler),
	)

	router.progressHandler, err = progresshttpdriver.New(
		config.NamespaceDecoder,
		config.ProgressManager,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("progress handler: %w", err)
	}

	// Customer
	router.customerHandler, err = customerhttpdriver.New(
		config.NamespaceDecoder,
		config.Customer,
		config.SubscriptionService,
		config.EntitlementConnector,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("customer handler: %w", err)
	}

	// App
	router.appHandler, err = apphttpdriver.New(
		config.Logger,
		config.NamespaceDecoder,
		config.App,
		config.AppStripe,
		config.Billing,
		config.Customer,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("app handler: %w", err)
	}

	// App Stripe
	router.appStripeHandler, err = appstripehttpdriver.New(
		config.NamespaceDecoder,
		config.AppStripe,
		config.Billing,
		config.Customer,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("app stripe handler: %w", err)
	}

	// App Custom Invoicing
	router.appCustomInvoicingHandler, err = appcustominvoicinghttpdriver.New(
		config.AppCustomInvoicing,
		config.NamespaceDecoder,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("app custom invoicing handler: %w", err)
	}

	// Billing
	router.billingHandler, err = billinghttpdriver.New(
		config.Logger,
		config.NamespaceDecoder,
		config.BillingFeatureSwitches,
		config.Billing,
		config.App,
		config.ChargeService,
		config.Credits,
		config.FeatureGate,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("billing handler: %w", err)
	}

	// Product Catalog
	router.planHandler, err = planhttpdriver.New(
		config.NamespaceDecoder,
		config.Plan,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("plan handler: %w", err)
	}

	router.addonHandler, err = addonhttpdriver.New(
		config.NamespaceDecoder,
		config.Addon,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("add-on handler: %w", err)
	}

	router.planAddonHandler, err = planaddonhttpdriver.New(
		config.NamespaceDecoder,
		config.PlanAddon,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("plan add-on handler: %w", err)
	}

	router.subjectHandler, err = subjecthttphandler.New(
		config.NamespaceDecoder,
		config.Logger,
		config.SubjectService,
		config.EntitlementConnector,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("subject handler: %w", err)
	}

	subscriptionHandler, err := subscriptionhttpdriver.NewHandler(
		subscriptionhttpdriver.HandlerConfig{
			SubscriptionWorkflowService: config.SubscriptionWorkflowService,
			SubscriptionService:         config.SubscriptionService,
			PlanSubscriptionService:     config.PlanSubscriptionService,
			NamespaceDecoder:            config.NamespaceDecoder,
			CustomerService:             config.Customer,
			Logger:                      config.Logger,
			Credits:                     config.Credits,
		},
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("subscription handler: %w", err)
	}

	router.subscriptionHandler = subscriptionHandler

	subscriptionAddonHandler, err := subscriptionaddonhttpdriver.NewHandler(
		subscriptionaddonhttpdriver.HandlerConfig{
			SubscriptionAddonService:    config.SubscriptionAddonService,
			SubscriptionWorkflowService: config.SubscriptionWorkflowService,
			SubscriptionService:         config.SubscriptionService,
			AddonService:                config.Addon,
			NamespaceDecoder:            config.NamespaceDecoder,
			Logger:                      config.Logger,
		},
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("subscription add-on handler: %w", err)
	}

	router.subscriptionAddonHandler = subscriptionAddonHandler

	// Portal
	router.portalHandler, err = portalhttphandler.New(
		config.NamespaceDecoder,
		config.Portal,
		config.MeterManageService,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("portal handler: %w", err)
	}

	// Currencies
	resolveNamespace := func(ctx context.Context) (string, error) {
		ns, ok := config.NamespaceDecoder.GetNamespace(ctx)
		if !ok {
			return "", v3apierrors.NewInternalError(ctx, errors.New("failed to resolve namespace"))
		}

		return ns, nil
	}

	router.currencyHandler, err = currencyhandler.New(
		resolveNamespace,
		config.CurrencyService,
		httptransport.WithErrorHandler(config.ErrorHandler),
	)
	if err != nil {
		return fmt.Errorf("currency handler: %w", err)
	}

	return nil
}
