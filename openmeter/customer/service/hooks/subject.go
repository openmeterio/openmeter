package hooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"strconv"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	SubjectHook     = models.ServiceHook[subject.Subject]
	NoopSubjectHook = models.NoopServiceHook[subject.Subject]
)

var _ models.ServiceHook[subject.Subject] = (*subjectHook)(nil)

type subjectHook struct {
	models.NoopServiceHook[subject.Subject]

	provisioner *CustomerProvisioner
	logger      *slog.Logger

	ignoreErrors bool
}

func (s subjectHook) provisionCustomerForSubject(ctx context.Context, sub *subject.Subject) error {
	err := s.provisioner.ProvisionCustomerForSubject(ctx, sub)
	if err != nil {
		if s.ignoreErrors {
			s.logger.Warn("failed to provision customer for subject", "error", err)

			return nil
		} else {
			s.logger.Error("failed to provision customer for subject", "error", err)

			return err
		}
	}

	return nil
}

func (s subjectHook) PostCreate(ctx context.Context, sub *subject.Subject) error {
	return s.provisionCustomerForSubject(ctx, sub)
}

func (s subjectHook) PostUpdate(ctx context.Context, sub *subject.Subject) error {
	return s.provisionCustomerForSubject(ctx, sub)
}

func NewSubjectHook(config SubjectHookConfig) (SubjectHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid subject hook config: %w", err)
	}

	provisioner, err := NewCustomerProvisioner(CustomerProvisionerConfig{
		Customer:         config.Customer,
		CustomerOverride: config.CustomerOverride,
		Logger:           config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize customer provisioner: %w", err)
	}

	return &subjectHook{
		provisioner:  provisioner,
		logger:       config.Logger.With("subsystem", "customer.subject.hook"),
		ignoreErrors: config.IgnoreErrors,
	}, nil
}

type SubjectHookConfig struct {
	Customer         customer.Service
	CustomerOverride billing.CustomerOverrideService
	Logger           *slog.Logger

	// IgnoreErrors if set to true makes the hooks ignore (not returning error)
	IgnoreErrors bool
}

func (c SubjectHookConfig) Validate() error {
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

	return errors.Join(errs...)
}

func CmpSubjectCustomer(s *subject.Subject, c *customer.Customer) bool {
	if c == nil || s == nil {
		return false
	}

	if c.Namespace != s.Namespace {
		return false
	}

	if !lo.Contains(c.UsageAttribution.SubjectKeys, s.Key) {
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
	}, nil
}

type CustomerProvisioner struct {
	customer         customer.Service
	customerOverride billing.CustomerOverrideService
	logger           *slog.Logger
}

var ErrCustomerKeyConflict = errors.New("customer key conflict")

func (p CustomerProvisioner) GetCustomerForSubject(ctx context.Context, sub *subject.Subject) (*customer.Customer, error) {
	var (
		cus *customer.Customer
		err error
	)

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

	// Return Customer if it has the Subject in usage attribution.
	// There are cases where the Customer and the Subject have the same key,
	// while the Subject is not included in the Customers usage attribution.
	// In this case the Customer must not match the Subject.
	if cus != nil {
		if lo.Contains(cus.UsageAttribution.SubjectKeys, sub.Key) {
			return cus, nil
		}

		return nil, ErrCustomerKeyConflict
	}

	// Try to find Customer for Subject by usage attribution
	return p.customer.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace:  sub.Namespace,
		SubjectKey: sub.Key,
	})
}

// EnsureCustomer returns a Customer entity created/updated based on the provided Subject.
func (p CustomerProvisioner) EnsureCustomer(ctx context.Context, sub *subject.Subject) (*customer.Customer, error) {
	if sub == nil {
		return nil, errors.New("failed to provision customer for subject: subject is nil")
	}

	var keyConflict bool

	cus, err := p.GetCustomerForSubject(ctx, sub)
	if err != nil {
		if errors.Is(err, ErrCustomerKeyConflict) {
			keyConflict = true
		} else if !models.IsGenericNotFoundError(err) {
			return nil, err
		}
	}

	annotations := models.Annotations{
		"createdBy": "customer.subject.hook",
		"subjectId": sub.Id,
	}

	if sub.StripeCustomerId != nil {
		annotations["stripeCustomerId"] = *sub.StripeCustomerId
	}

	if cus != nil {
		if CmpSubjectCustomer(sub, cus) {
			return cus, nil
		}

		// Update Customer for Subject in case there is non to be found
		cus, err = p.customer.UpdateCustomer(ctx, customer.UpdateCustomerInput{
			CustomerID: customer.CustomerID{
				Namespace: cus.Namespace,
				ID:        cus.ID,
			},
			CustomerMutate: customer.CustomerMutate{
				Key:              lo.ToPtr(lo.FromPtrOr(cus.Key, sub.Key)),
				Name:             lo.FromPtr(sub.DisplayName),
				Description:      cus.Description,
				UsageAttribution: cus.UsageAttribution,
				PrimaryEmail:     cus.PrimaryEmail,
				Currency:         cus.Currency,
				BillingAddress:   cus.BillingAddress,
				Metadata:         lo.ToPtr(cus.Metadata.Merge(MetadataFromMap(sub.Metadata))),
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
				cus.Namespace, cus.ID, err)
		}

		return cus, nil
	}

	// Create Customer for Subject in case there is none to be found
	return p.customer.CreateCustomer(ctx, customer.CreateCustomerInput{
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
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: []string{sub.Key},
			},
			PrimaryEmail:   nil,
			Currency:       nil,
			BillingAddress: nil,
			Metadata:       lo.ToPtr(MetadataFromMap(sub.Metadata)),
			Annotation:     &annotations,
		},
	})
}

func (p CustomerProvisioner) EnsureStripeCustomer(ctx context.Context, customerID customer.CustomerID, stripeCustomerID string) error {
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

	if profile.Apps == nil {
		return fmt.Errorf("failed to setup stripe customer id for customer [namespace=%s customer.id=%s]: apps profile is nil",
			customerID.Namespace, customerID.ID)
	}

	if appPaymentType := profile.Apps.Payment.GetType(); appPaymentType != app.AppTypeStripe {
		return fmt.Errorf("failed to setup stripe customer id for customer [namespace=%s customer.id=%s app.payment.type=%s]: payment app is not stripe",
			customerID.Namespace, customerID.ID, appPaymentType)
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

	return nil
}

func (p CustomerProvisioner) ProvisionCustomerForSubject(ctx context.Context, sub *subject.Subject) error {
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
		if err := p.EnsureStripeCustomer(ctx, customerID, *sub.StripeCustomerId); err != nil {
			return fmt.Errorf("failed to update stripe customer id for subject customer [namespace=%s subject.id=%s subject.key=%s customer.id=%s]: %w",
				sub.Namespace, sub.Id, sub.Key, cus.ID, err)
		}

		return nil
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
