package linerouter

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ billing.CreateLineRouter = (*Router)(nil)

type Config struct {
	CreditsEnabled           bool
	CreditThenInvoiceEnabled bool
	FeatureGate              *featuregate.FeatureGateChecker
}

func (c Config) Validate() error {
	var errs []error

	if c.FeatureGate == nil {
		errs = append(errs, errors.New("feature gate is required"))
	} else if err := c.FeatureGate.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature gate: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Router struct {
	creditsEnabled           bool
	creditThenInvoiceEnabled bool
	featureGate              *featuregate.FeatureGateChecker
}

func New(config Config) (*Router, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Router{
		creditsEnabled:           config.CreditsEnabled,
		creditThenInvoiceEnabled: config.CreditThenInvoiceEnabled,
		featureGate:              config.FeatureGate,
	}, nil
}

func (r *Router) GetLineEngineForCreateLine(line billing.GenericInvoiceLineReader) (billing.LineEngineType, error) {
	if line == nil {
		return "", fmt.Errorf("line is required")
	}

	available, err := r.chargesAvailable(line)
	if err != nil {
		return "", err
	}

	if !available {
		return billing.LineEngineTypeInvoice, nil
	}

	return lineEngineFromPrice(line)
}

func (r *Router) chargesAvailable(line billing.GenericInvoiceLineReader) (bool, error) {
	if !r.creditsEnabled || !r.creditThenInvoiceEnabled {
		return false, nil
	}

	namespace := line.GetLineID().Namespace
	if namespace == "" {
		return false, fmt.Errorf("line[%s]: namespace is required", line.GetID())
	}

	return r.featureGate.Enabled(namespace, r.featureGate.Flags.Credits())
}

func lineEngineFromPrice(line billing.GenericInvoiceLineReader) (billing.LineEngineType, error) {
	price := line.GetPrice()
	if price == nil {
		return "", fmt.Errorf("line[%s]: price is required", line.GetID())
	}

	switch price.Type() {
	case productcatalog.FlatPriceType:
		return billing.LineEngineTypeChargeFlatFee, nil
	default:
		return billing.LineEngineTypeChargeUsageBased, nil
	}
}
