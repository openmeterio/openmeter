package ffx

import (
	"context"
	"fmt"
)

type contextKey string

const (
	accessContextKey contextKey = "access"
)

var ContextMissingAccessError = fmt.Errorf("access not found in context")

func SetAccessOnContext(ctx context.Context, access AccessConfig) context.Context {
	return context.WithValue(ctx, accessContextKey, access)
}

func GetAccessFromContext(ctx context.Context) (AccessConfig, error) {
	access, ok := ctx.Value(accessContextKey).(AccessConfig)
	if !ok {
		return nil, ContextMissingAccessError
	}
	return access, nil
}

type contextService struct{}

func (s *contextService) IsFeatureEnabled(ctx context.Context, feature Feature) (bool, error) {
	access, err := GetAccessFromContext(ctx)
	if err != nil {
		return false, err
	}

	acc, ok := access[feature]
	if !ok {
		return false, fmt.Errorf("feature %s not found in access", feature)
	}

	return acc, nil
}

func NewContextService() Service {
	return &contextService{}
}
