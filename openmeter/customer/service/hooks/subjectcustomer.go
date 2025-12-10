package hooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectservicehooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	SubjectCustomerHook     = models.ServiceHook[subject.Subject]
	NoopSubjectCustomerHook = models.NoopServiceHook[subject.Subject]
)

var _ models.ServiceHook[subject.Subject] = (*subjectCustomerHook)(nil)

type subjectCustomerHook struct {
	NoopSubjectCustomerHook

	provisioner *CustomerProvisioner
	logger      *slog.Logger
	tracer      trace.Tracer

	ignoreErrors bool
}

func (s subjectCustomerHook) provision(ctx context.Context, sub *subject.Subject) error {
	err := s.provisioner.Provision(ctx, sub)
	if err != nil {
		if s.ignoreErrors {
			s.logger.WarnContext(ctx, "failed to provision customer for subject", "error", err)

			return nil
		}

		return err
	}

	return nil
}

func (s subjectCustomerHook) PostDelete(ctx context.Context, sub *subject.Subject) error {
	ctx, span := s.tracer.Start(ctx, "subject_customer_hook.post_delete", trace.WithAttributes(
		attribute.String("subject.id", sub.Id),
		attribute.String("subject.key", sub.Key),
	))
	defer span.End()

	// Let's get the customer by usage attribution
	cus, err := s.provisioner.customer.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: sub.Namespace,
		Key:       sub.Key,
	})
	if err != nil {
		if models.IsGenericNotFoundError(err) {
			span.AddEvent("customer not found by usage attribution", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))

			return nil
		}

		return err
	}

	if cus == nil {
		span.AddEvent("customer not found by usage attribution")

		return nil
	}

	if cus.DeletedAt != nil && cus.DeletedAt.Before(clock.Now()) {
		span.AddEvent("customer is deleted", trace.WithAttributes(
			attribute.String("customer.id", cus.ID),
			attribute.String("customer.deleted_at", cus.DeletedAt.Format(time.RFC3339)),
		))

		return nil
	}

	// Let's update the customer usage attribution
	cus, err = s.provisioner.customer.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: customer.CustomerID{
			Namespace: cus.Namespace,
			ID:        cus.ID,
		},
		CustomerMutate: func() customer.CustomerMutate {
			mut := cus.AsCustomerMutate()

			if mut.UsageAttribution != nil {
				mut.UsageAttribution.SubjectKeys = lo.Filter(mut.UsageAttribution.SubjectKeys, func(key string, _ int) bool {
					return key != sub.Key
				})
			}

			return mut
		}(),
	})

	if cus != nil {
		var subjectKeysStr string
		if cus.UsageAttribution != nil {
			subjectKeysStr = strings.Join(cus.UsageAttribution.SubjectKeys, ", ")
		}
		span.AddEvent("updated customer usage attribution", trace.WithAttributes(
			attribute.String("customer.usage_attribution.subject_keys", subjectKeysStr),
		))
	}

	if err != nil {
		span.AddEvent("failed to update customer usage attribution", trace.WithAttributes(
			attribute.String("error", err.Error()),
		))

		return err
	}

	return nil
}

func (s subjectCustomerHook) PostCreate(ctx context.Context, sub *subject.Subject) error {
	ctx, span := s.tracer.Start(ctx, "subject_customer_hook.post_create")
	defer span.End()

	err := s.provision(ctx, sub)
	if err != nil {
		span.SetStatus(otelcodes.Error, "failed to provision customer for subject")
		span.RecordError(err)
	} else {
		span.SetStatus(otelcodes.Ok, "customer provisioned for subject")
	}

	return err
}

func (s subjectCustomerHook) PostUpdate(ctx context.Context, sub *subject.Subject) error {
	ctx, span := s.tracer.Start(ctx, "subject_customer_hook.post_update")
	defer span.End()

	err := s.provision(ctx, sub)
	if err != nil {
		span.SetStatus(otelcodes.Error, "failed to provision customer for subject")
		span.RecordError(err)
	} else {
		span.SetStatus(otelcodes.Ok, "customer provisioned for subject")
	}

	return err
}

func NewSubjectCustomerHook(config SubjectCustomerHookConfig) (SubjectCustomerHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subject hook config: %w", err)
	}

	provisioner, err := NewCustomerProvisioner(CustomerProvisionerConfig{
		Customer:         config.Customer,
		CustomerOverride: config.CustomerOverride,
		Logger:           config.Logger,
		Tracer:           config.Tracer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize customer provisioner: %w", err)
	}

	return &subjectCustomerHook{
		provisioner:  provisioner,
		logger:       config.Logger.With("subsystem", "subject_customer_provisioner"),
		tracer:       config.Tracer,
		ignoreErrors: config.IgnoreErrors,
	}, nil
}

type SubjectCustomerHookConfig struct {
	Customer         customer.Service
	CustomerOverride billing.CustomerOverrideService
	Logger           *slog.Logger
	Tracer           trace.Tracer

	// IgnoreErrors if set to true makes the hooks ignore (not returning error)
	IgnoreErrors bool
}

func (c SubjectCustomerHookConfig) Validate() error {
	var errs []error

	if c.Customer == nil {
		errs = append(errs, fmt.Errorf("customer service is required"))
	}

	if c.CustomerOverride == nil {
		errs = append(errs, fmt.Errorf("customer override service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, fmt.Errorf("logger is required"))
	}

	if c.Tracer == nil {
		errs = append(errs, fmt.Errorf("tracer is required"))
	}

	return errors.Join(errs...)
}

func CmpSubjectCustomer(s *subject.Subject, c *customer.Customer) bool {
	if c == nil || s == nil {
		return false
	}

	if c.Namespace != s.Namespace {
		return false
	}

	var subjectKeys []string
	if c.UsageAttribution != nil {
		subjectKeys = c.UsageAttribution.SubjectKeys
	}

	if !lo.Contains(subjectKeys, s.Key) {
		return false
	}

	if c.Key == nil {
		return false
	}

	if s.DisplayName != nil && *s.DisplayName != c.Name {
		return false
	}

	sm := MetadataFromMap(s.Metadata)
	cm := lo.FromPtr(c.Metadata)
	for k, v := range sm {
		if vv, ok := cm[k]; !ok || vv != v {
			return false
		}
	}

	return true
}

var _ models.Validator = (*CustomerProvisionerConfig)(nil)

type CustomerProvisionerConfig struct {
	Customer         customer.Service
	CustomerOverride billing.CustomerOverrideService
	Logger           *slog.Logger
	Tracer           trace.Tracer
}

func (c CustomerProvisionerConfig) Validate() error {
	var errs []error

	if c.Customer == nil {
		errs = append(errs, fmt.Errorf("customer service is required"))
	}

	if c.CustomerOverride == nil {
		errs = append(errs, fmt.Errorf("customer override service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, fmt.Errorf("logger is required"))
	}

	if c.Tracer == nil {
		errs = append(errs, fmt.Errorf("tracer is required"))
	}

	return errors.Join(errs...)
}

func NewCustomerProvisioner(config CustomerProvisionerConfig) (*CustomerProvisioner, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid customer provisioner config: %w", err)
	}

	return &CustomerProvisioner{
		customer:         config.Customer,
		customerOverride: config.CustomerOverride,
		logger:           config.Logger.With("subsystem", "customer.provisioner"),
		tracer:           config.Tracer,
	}, nil
}

type CustomerProvisioner struct {
	customer         customer.Service
	customerOverride billing.CustomerOverrideService
	logger           *slog.Logger
	tracer           trace.Tracer
}

var ErrCustomerKeyConflict = errors.New("customer key conflict")

func (p CustomerProvisioner) getCustomerForSubject(ctx context.Context, sub *subject.Subject) (*customer.Customer, error) {
	// Try to find Customer for Subject by usage attribution
	cus, err := p.customer.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: sub.Namespace,
		Key:       sub.Key,
	})
	if err != nil && !models.IsGenericNotFoundError(err) {
		return nil, err
	}

	if cus != nil && cus.DeletedAt == nil {
		return cus, nil
	}

	// Try to find Customer for Subject by key
	cus, err = p.customer.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerKey: &customer.CustomerKey{
			Namespace: sub.Namespace,
			Key:       sub.Key,
		},
	})
	if err != nil && !models.IsGenericNotFoundError(err) {
		return nil, err
	}

	if cus != nil && cus.IsDeleted() {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
		)
	}

	// Return Customer if it has the Subject in usage attribution.
	// There are cases where the Customer and the Subject have the same key,
	// while the Subject is not included in the Customers usage attribution.
	// In this case the Customer must not match the Subject.
	if cus != nil && cus.DeletedAt == nil {
		var subjectKeys []string
		if cus.UsageAttribution != nil {
			subjectKeys = cus.UsageAttribution.SubjectKeys
		}

		if lo.Contains(subjectKeys, sub.Key) {
			return cus, nil
		}

		return nil, models.NewGenericConflictError(ErrCustomerKeyConflict)
	}

	return nil, models.NewGenericNotFoundError(fmt.Errorf("failed to find customer for subject [namespace=%s subject.key=%s]",
		sub.Namespace, sub.Key))
}

// EnsureCustomer returns a Customer entity created/updated based on the provided Subject.
func (p CustomerProvisioner) EnsureCustomer(ctx context.Context, sub *subject.Subject) (*customer.Customer, error) {
	if sub == nil {
		return nil, errors.New("failed to provision customer for subject: subject is nil")
	}

	var err error

	ctx, span := p.tracer.Start(ctx, "customer_provisioner.ensure_customer")
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(otelcodes.Ok, "customer provisioned for subject")
		}

		span.End()
	}()

	span.SetAttributes(
		attribute.String("subject.id", sub.Id),
		attribute.String("subject.key", sub.Key),
		attribute.String("subject.stripe_customer_id", lo.FromPtrOr(sub.StripeCustomerId, "nil")),
	)

	var keyConflict bool

	cus, err := p.getCustomerForSubject(ctx, sub)
	if err != nil {
		if errors.Is(err, ErrCustomerKeyConflict) {
			keyConflict = true
		} else if !models.IsGenericNotFoundError(err) {
			return nil, err
		}
	}

	// Ignore deleted Customers
	if cus != nil && cus.DeletedAt != nil {
		cus = nil
		keyConflict = false
	}

	annotations := models.Annotations{
		"createdBy": "customer.provisioner",
		"subjectId": sub.Id,
	}

	if sub.StripeCustomerId != nil {
		annotations["stripeCustomerId"] = *sub.StripeCustomerId
	}

	if cus != nil {
		span.AddEvent("found customer", trace.WithAttributes(
			attribute.String("customer.id", cus.ID),
			attribute.String("customer.key", lo.FromPtrOr(cus.Key, "nil")),
		))

		if CmpSubjectCustomer(sub, cus) {
			return cus, nil
		}

		customerID := customer.CustomerID{
			Namespace: cus.Namespace,
			ID:        cus.ID,
		}

		// Update Customer for Subject in case there is non to be found
		cus, err = p.customer.UpdateCustomer(
			subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx),
			customer.UpdateCustomerInput{
				CustomerID: customerID,
				CustomerMutate: customer.CustomerMutate{
					Key:              cus.Key,
					Name:             lo.FromPtrOr(sub.DisplayName, cus.Name),
					Description:      cus.Description,
					UsageAttribution: cus.UsageAttribution,
					PrimaryEmail:     cus.PrimaryEmail,
					Currency:         cus.Currency,
					BillingAddress:   cus.BillingAddress,
					Metadata: func() *models.Metadata {
						cm := lo.FromPtr(cus.Metadata)

						if len(sub.Metadata) == 0 && len(cm) == 0 {
							return nil
						}

						return lo.ToPtr(cm.Merge(MetadataFromMap(sub.Metadata)))
					}(),
					Annotation: func() *models.Annotations {
						if len(lo.FromPtr(cus.Annotation)) == 0 && len(annotations) == 0 {
							return nil
						}

						m := make(models.Annotations)

						maps.Copy(m, lo.FromPtr(cus.Annotation))
						maps.Copy(m, annotations)

						return &m
					}(),
				},
			})
		if err != nil {
			return nil, fmt.Errorf("failed to update customer for subject [namespace=%s customer.id=%s]: %w",
				customerID.Namespace, customerID.ID, err)
		}

		span.AddEvent("updated customer")

		return cus, nil
	}

	// Create Customer for Subject in case there is none to be found
	cus, err = p.customer.CreateCustomer(
		subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx),
		customer.CreateCustomerInput{
			Namespace: sub.Namespace,
			CustomerMutate: customer.CustomerMutate{
				Key: func() *string {
					if keyConflict {
						return nil
					}

					return lo.ToPtr(sub.Key)
				}(),
				Name:        lo.FromPtrOr(sub.DisplayName, sub.Key),
				Description: nil,
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{sub.Key},
				},
				PrimaryEmail:   nil,
				Currency:       nil,
				BillingAddress: nil,
				Metadata:       lo.ToPtr(MetadataFromMap(sub.Metadata)),
				Annotation:     &annotations,
			},
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer for subject [namespace=%s subject.key=%s]: %w",
			sub.Namespace, sub.Key, err)
	}

	span.AddEvent("created customer", trace.WithAttributes(
		attribute.String("customer.id", cus.ID),
		attribute.String("customer.key", lo.FromPtrOr(cus.Key, "nil")),
	))

	return cus, err
}

type InvalidPaymentAppError struct {
	AppType app.AppType
	AppID   app.AppID
}

func (e InvalidPaymentAppError) Error() string {
	return fmt.Sprintf("invalid payment app type [app.type=%s app.id=%s]", e.AppType, e.AppID.ID)
}

func (p CustomerProvisioner) EnsureStripeCustomer(ctx context.Context, customerID customer.CustomerID, stripeCustomerID string) error {
	var err error

	ctx, span := p.tracer.Start(ctx, "customer_provisioner.ensure_stripe_customer")
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(otelcodes.Ok, "stripe customer provisioned")
		}

		span.End()
	}()

	span.SetAttributes(
		attribute.String("customer.id", customerID.ID),
		attribute.String("customer.namespace", customerID.Namespace),
	)

	customerOverride, err := p.customerOverride.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customerID,
		Expand: billing.CustomerOverrideExpand{
			Apps: true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get customer override for subject [namespace=%s customer.id=%s]: %w",
			customerID.Namespace, customerID.ID, err)
	}

	profile := customerOverride.MergedProfile

	span.AddEvent("fetched customer billing profile", trace.WithAttributes(
		attribute.String("profile.id", profile.ID),
		attribute.String("profile.namespace", profile.Namespace),
	))

	if profile.Apps == nil {
		return fmt.Errorf("failed to setup stripe customer id for customer [namespace=%s customer.id=%s]: apps profile is nil",
			customerID.Namespace, customerID.ID)
	}

	if appPaymentType := profile.Apps.Payment.GetType(); appPaymentType != app.AppTypeStripe {
		return InvalidPaymentAppError{
			AppType: appPaymentType,
			AppID:   profile.Apps.Payment.GetID(),
		}
	}

	err = profile.Apps.Payment.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customerID,
		Data: appstripeentity.CustomerData{
			StripeCustomerID: stripeCustomerID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to setup stripe customer id for customer [namespace=%s customer.id=%s]: %w",
			customerID.Namespace, customerID.ID, err)
	}

	span.AddEvent("updated stripe customer data", trace.WithAttributes(
		attribute.String("app.id", profile.Apps.Payment.GetID().ID),
		attribute.String("app.namespace", profile.Apps.Payment.GetID().Namespace),
	))

	return nil
}

func (p CustomerProvisioner) Provision(ctx context.Context, sub *subject.Subject) error {
	if subjectservicehooks.SkipSubjectCustomerFromContext(ctx) {
		return nil
	}

	cus, err := p.EnsureCustomer(ctx, sub)
	if err != nil {
		return fmt.Errorf("failed to provision customer for subject [namespace=%s subject.id=%s subject.key=%s]: %w",
			sub.Namespace, sub.Id, sub.Key, err)
	}

	customerID := customer.CustomerID{
		Namespace: cus.Namespace,
		ID:        cus.ID,
	}

	if sub.StripeCustomerId != nil {
		err = p.EnsureStripeCustomer(ctx, customerID, *sub.StripeCustomerId)
		if err != nil {
			err = fmt.Errorf("failed to set stripe customer id for subject customer [namespace=%s subject.id=%s subject.key=%s customer.id=%s]: %w",
				sub.Namespace, sub.Id, sub.Key, cus.ID, err)

			// Ignore InvalidPaymentAppError by logging it on the warning level otherwise return the error.
			var invalidErr InvalidPaymentAppError

			if !errors.As(err, &invalidErr) {
				return err
			}

			p.logger.WarnContext(ctx, err.Error())
		}
	}

	return nil
}

func MetadataFromMap(m map[string]interface{}) models.Metadata {
	if len(m) == 0 {
		return nil
	}

	var metadata models.Metadata

	for k, v := range m {
		if value := toString(v, 0); value != "" {
			if metadata == nil {
				metadata = make(models.Metadata)
			}

			metadata[k] = value
		}
	}

	return metadata
}

func toString(v interface{}, rec int) string {
	if v == nil {
		return ""
	}

	if rec > 1 {
		return ""
	}

	vv := reflect.ValueOf(v)

	switch vv.Kind() {
	case reflect.Ptr:
		if vv.IsNil() {
			return ""
		}

		return toString(vv.Elem().Interface(), rec+1)
	case reflect.Map:
		if vv.Len() == 0 {
			return ""
		}

		var result []string
		for _, k := range vv.MapKeys() {
			if k.Kind() != reflect.String {
				continue
			}

			if s := toString(vv.MapIndex(k).Interface(), rec+1); s != "" {
				result = append(result, `"`+k.String()+`"="`+s+`"`)
			}
		}

		slices.Sort(result)

		return strings.Join(result, ",")
	case reflect.Slice:
		if vv.Len() == 0 {
			return ""
		}

		var result []string
		for i := 0; i < vv.Len(); i++ {
			if s := toString(vv.Index(i).Interface(), rec+1); s != "" {
				result = append(result, `"`+s+`"`)
			}
		}

		return strings.Join(result, ",")
	case reflect.String:
		return v.(string)
	case reflect.Int:
		return strconv.FormatInt(int64(v.(int)), 10)
	case reflect.Int8:
		return strconv.FormatInt(int64(v.(int8)), 10)
	case reflect.Int16:
		return strconv.FormatInt(int64(v.(int16)), 10)
	case reflect.Int32:
		return strconv.FormatInt(int64(v.(int32)), 10)
	case reflect.Int64:
		return strconv.FormatInt(v.(int64), 10)
	case reflect.Uint:
		return strconv.FormatUint(uint64(v.(uint)), 10)
	case reflect.Uint8:
		return strconv.FormatUint(uint64(v.(uint8)), 10)
	case reflect.Uint16:
		return strconv.FormatUint(uint64(v.(uint16)), 10)
	case reflect.Uint32:
		return strconv.FormatUint(uint64(v.(uint32)), 10)
	case reflect.Uint64:
		return strconv.FormatUint(v.(uint64), 10)
	case reflect.Float32:
		return strconv.FormatFloat(float64(v.(float32)), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.(bool))
	default:
		return ""
	}
}
