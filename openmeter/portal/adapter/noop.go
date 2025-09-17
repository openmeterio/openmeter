package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type noopAdapter struct{}

func NewNoop() portal.Service {
	return &noopAdapter{}
}

func (a *noopAdapter) CreateToken(ctx context.Context, input portal.CreateTokenInput) (*portal.PortalToken, error) {
	return nil, models.NewGenericNotImplementedError(fmt.Errorf("noop adapter"))
}

func (a *noopAdapter) Validate(ctx context.Context, tokenString string) (*portal.PortalTokenClaims, error) {
	return nil, models.NewGenericNotImplementedError(fmt.Errorf("noop adapter"))
}

func (a *noopAdapter) ListTokens(ctx context.Context, input portal.ListTokensInput) (pagination.Result[*portal.PortalToken], error) {
	return pagination.Result[*portal.PortalToken]{}, models.NewGenericNotImplementedError(fmt.Errorf("noop adapter"))
}

func (a *noopAdapter) InvalidateToken(ctx context.Context, input portal.InvalidateTokenInput) error {
	return models.NewGenericNotImplementedError(fmt.Errorf("noop adapter"))
}
