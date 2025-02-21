package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type noopAdapter struct{}

func NewNoop() portal.Service {
	return &noopAdapter{}
}

func (a *noopAdapter) CreateToken(ctx context.Context, input portal.CreateTokenInput) (*portal.PortalToken, error) {
	return nil, portal.NewNotImplementedError(fmt.Errorf("not implemented"))
}

func (a *noopAdapter) Validate(tokenString string) (*portal.PortalTokenClaims, error) {
	return nil, portal.NewNotImplementedError(fmt.Errorf("not implemented"))
}

func (a *noopAdapter) ListTokens(ctx context.Context, input portal.ListTokensInput) (pagination.PagedResponse[*portal.PortalToken], error) {
	return pagination.PagedResponse[*portal.PortalToken]{}, portal.NewNotImplementedError(fmt.Errorf("not implemented"))
}

func (a *noopAdapter) InvalidateToken(ctx context.Context, input portal.InvalidateTokenInput) error {
	return portal.NewNotImplementedError(fmt.Errorf("not implemented"))
}
