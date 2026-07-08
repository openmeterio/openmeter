package creditvoid

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
)

// NoopService is wired when credits are disabled.
type NoopService struct{}

var _ Service = NoopService{}

func NewNoopService() Service {
	return NoopService{}
}

func (NoopService) VoidCreditPurchase(context.Context, VoidCreditPurchaseInput) (VoidCreditPurchaseResult, error) {
	return VoidCreditPurchaseResult{}, models.NewGenericNotImplementedError(errors.New("credit voiding is not enabled"))
}

func (NoopService) ListVoidedCreditImpacts(context.Context, ListVoidedCreditImpactsInput) (ListVoidedCreditImpactsResult, error) {
	return ListVoidedCreditImpactsResult{Items: []VoidImpact{}}, nil
}
