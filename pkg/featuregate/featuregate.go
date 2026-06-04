package featuregate

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
)

type Gate interface {
	EvaluateBool(namespace, flag string, defaultValue bool) (bool, error)
}

func NewNoop() Gate {
	return Noop{}
}

type Noop struct{}

func (n Noop) EvaluateBool(string, string, bool) (bool, error) {
	return true, nil
}

var _ fmt.Stringer = (*FeatureFlag)(nil)

type FeatureFlag string

func (f FeatureFlag) String() string {
	return string(f)
}

const (
	credits FeatureFlag = "om_ff_credits_enabled"
)

func ContextResolver() contextResolver {
	return contextResolver{}
}

type contextResolver struct{}

func (r contextResolver) Credits(ctx context.Context) (bool, bool) {
	value, found := ctx.Value(credits).(bool)
	return value, found
}

type Flags map[FeatureFlag]string

func (f *Flags) Keys() []FeatureFlag {
	return []FeatureFlag{credits}
}

func (f *Flags) Validate() error {
	if f == nil || len(*f) == 0 {
		return errors.New("featuregate is enabled but missing flags setup")
	}
	keys := f.Keys()

	for k := range *f {
		if !slices.Contains(keys, k) {
			return fmt.Errorf("invalid key: %s", k)
		}
	}

	return nil
}

func (f *Flags) Credits() string {
	if f == nil {
		return ""
	}
	value, ok := (*f)[credits]
	if !ok {
		return ""
	}
	return value
}

const defaultCacheSize = 1024

func NewFeatureGateChecker(gate Gate, flags Flags) *FeatureGateChecker {
	checker := &FeatureGateChecker{
		Gate:  gate,
		Flags: flags,
	}
	cacheSize := defaultCacheSize

	var err error
	checker.store, err = lru.New[string, bool](cacheSize)
	if err != nil {
		return checker
	}

	return checker
}

type FeatureGateChecker struct {
	Gate  Gate
	Flags Flags

	store *lru.Cache[string, bool]
}

func (h *FeatureGateChecker) Validate() error {
	if h == nil || h.Gate == nil {
		return errors.New("feature gate is required")
	}

	return nil
}

func (h *FeatureGateChecker) Enabled(ns string, flag string) (bool, error) {
	if h == nil {
		return true, nil
	}
	if h.Gate == nil {
		return true, nil
	}
	if flag == "" {
		return true, nil
	}

	cacheKey := strings.Join([]string{flag, ns}, "_")

	flagResult, cached := h.getFromCache(cacheKey)
	if !cached {
		enabled, err := h.Gate.EvaluateBool(ns, flag, false)
		if err != nil {
			return false, err
		}
		h.addToCache(cacheKey, enabled)
		return enabled, nil
	}

	return flagResult, nil
}

// getFromCache supposed to make cache fault tolerant
// so if store is not initialized, we return cache false
func (h FeatureGateChecker) getFromCache(key string) (bool, bool) {
	if h.store == nil {
		return false, false
	}

	return h.store.Get(key)
}

// addToCache supposed to make cache fault tolerant
// so if store is not initialized, we do an early exit
func (h FeatureGateChecker) addToCache(key string, value bool) {
	if h.store == nil {
		return
	}

	h.store.Add(key, value)
}

func NewMiddleware[Request any, Response any](GetNamespace func(ctx context.Context) (string, bool), checker *FeatureGateChecker) operation.Middleware[Request, Response] {
	return func(next operation.Operation[Request, Response]) operation.Operation[Request, Response] {
		return func(ctx context.Context, request Request) (Response, error) {
			ns, ok := GetNamespace(ctx)
			if !ok {
				return lo.Empty[Response](), commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
			}

			for contextFlagKey := range checker.Flags {
				configFlagKey := checker.Flags[contextFlagKey]
				result, err := checker.Enabled(ns, configFlagKey)
				if err != nil {
					return lo.Empty[Response](), err
				}
				ctx = context.WithValue(ctx, contextFlagKey, result)
			}

			return next(ctx, request)
		}
	}
}
