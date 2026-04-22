package run

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Service owns usage-based realization run mechanics: rating snapshots,
// run persistence, credit allocation/correction, and credit-realization lineage.
// It must not make state-machine decisions such as which triggers to fire, which
// statuses to enter, or whether invoice lifecycle events should advance a charge.
type Service struct {
	adapter usagebased.Adapter
	rater   usagebasedrating.Service
	handler usagebased.Handler
	lineage lineage.Service
}

type Config struct {
	Adapter usagebased.Adapter
	Rater   usagebasedrating.Service
	Handler usagebased.Handler
	Lineage lineage.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Rater == nil {
		errs = append(errs, errors.New("rater is required"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("handler is required"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
		rater:   config.Rater,
		handler: config.Handler,
		lineage: config.Lineage,
	}, nil
}

type CreditAllocationMode string

const (
	// CreditAllocationNone means no credits are allocated to the run.
	CreditAllocationNone CreditAllocationMode = "none"
	// CreditAllocationExact means the total's exact amount of credits is allocated to the run.
	CreditAllocationExact CreditAllocationMode = "exact"
	// CreditAllocationAvailable means credits should be allocated up to the total's amount of credits, but it's not an
	// error if the total's amount of credits is not available.
	CreditAllocationAvailable CreditAllocationMode = "available"
)

func (m CreditAllocationMode) Validate() error {
	switch m {
	case CreditAllocationNone, CreditAllocationExact, CreditAllocationAvailable:
		return nil
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid credit allocation mode: %s", m))
	}
}
