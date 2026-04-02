package service

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type service struct {
	adapter charges.Adapter
	// Note: if meta has a service layer, we should use it here instead of the adapter
	metaAdapter    meta.Adapter
	billingService billing.Service
	featureService feature.FeatureConnector

	flatFeeService        flatfee.Service
	creditPurchaseService creditpurchase.Service
	usageBasedService     usagebased.Service

	fsNamespaceLockdown []string
}

type Config struct {
	Adapter     charges.Adapter
	MetaAdapter meta.Adapter

	FeatureService        feature.FeatureConnector
	FlatFeeService        flatfee.Service
	CreditPurchaseService creditpurchase.Service
	UsageBasedService     usagebased.Service

	BillingService billing.Service

	FSNamespaceLockdown []string
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.BillingService == nil {
		errs = append(errs, errors.New("billing service cannot be null"))
	}

	if c.FeatureService == nil {
		errs = append(errs, errors.New("feature service cannot be null"))
	}

	if c.FlatFeeService == nil {
		errs = append(errs, errors.New("flat fee service cannot be null"))
	}

	if c.CreditPurchaseService == nil {
		errs = append(errs, errors.New("credit purchase service cannot be null"))
	}

	if c.UsageBasedService == nil {
		errs = append(errs, errors.New("usage based service cannot be null"))
	}

	if c.MetaAdapter == nil {
		errs = append(errs, errors.New("meta adapter cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	svc := &service{
		adapter:               config.Adapter,
		billingService:        config.BillingService,
		featureService:        config.FeatureService,
		metaAdapter:           config.MetaAdapter,
		flatFeeService:        config.FlatFeeService,
		creditPurchaseService: config.CreditPurchaseService,
		usageBasedService:     config.UsageBasedService,
		fsNamespaceLockdown:   config.FSNamespaceLockdown,
	}

	standardInvoiceEventHandler := &standardInvoiceEventHandler{
		chargesService: svc,
	}

	config.BillingService.RegisterStandardInvoiceHooks(standardInvoiceEventHandler)

	return svc, nil
}

func (s *service) validateNamespaceLockdown(namespace string) error {
	if slices.Contains(s.fsNamespaceLockdown, namespace) {
		return billing.ValidationError{
			Err: fmt.Errorf("%w: %s", billing.ErrNamespaceLocked, namespace),
		}
	}

	return nil
}

var _ charges.Service = (*service)(nil)
