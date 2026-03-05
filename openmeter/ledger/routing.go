package ledger

import (
	"errors"
	"fmt"
	"strings"
)

type RoutingKeyVersion string

const RoutingKeyVersionV1 RoutingKeyVersion = "v1"

func (v RoutingKeyVersion) Validate() error {
	switch v {
	case RoutingKeyVersionV1:
		return nil
	default:
		return fmt.Errorf("invalid routing key version: %s", v)
	}
}

type RoutingKey struct {
	version RoutingKeyVersion
	value   string
}

func NewRoutingKey(version RoutingKeyVersion, value string) (RoutingKey, error) {
	if err := version.Validate(); err != nil {
		return RoutingKey{}, err
	}
	if value == "" {
		return RoutingKey{}, errors.New("routing key is required")
	}

	return RoutingKey{
		version: version,
		value:   value,
	}, nil
}

func (k RoutingKey) Version() RoutingKeyVersion {
	return k.version
}

func (k RoutingKey) Value() string {
	return k.value
}

func MustNewRoutingKey(version RoutingKeyVersion, value string) RoutingKey {
	key, err := NewRoutingKey(version, value)
	if err != nil {
		panic(err)
	}
	return key
}

type SubAccountRoute struct {
	id  string
	key RoutingKey
}

func NewSubAccountRoute(id string, key RoutingKey) (SubAccountRoute, error) {
	if id == "" {
		return SubAccountRoute{}, errors.New("route id is required")
	}

	return SubAccountRoute{
		id:  id,
		key: key,
	}, nil
}

func MustNewSubAccountRoute(id string, key RoutingKey) SubAccountRoute {
	route, err := NewSubAccountRoute(id, key)
	if err != nil {
		panic(err)
	}
	return route
}

func (r SubAccountRoute) ID() string {
	return r.id
}

func (r SubAccountRoute) RoutingKey() RoutingKey {
	return r.key
}

type SubAccountRouteInput struct {
	CurrencyDimensionID       string
	TaxCodeDimensionID        *string
	FeaturesDimensionID       *string
	CreditPriorityDimensionID *string
}

func (i SubAccountRouteInput) Validate() error {
	if i.CurrencyDimensionID == "" {
		return errors.New("currency dimension id is required")
	}
	return nil
}

func BuildRoutingKey(version RoutingKeyVersion, input SubAccountRouteInput) (RoutingKey, error) {
	if err := input.Validate(); err != nil {
		return RoutingKey{}, err
	}

	switch version {
	case RoutingKeyVersionV1:
		return BuildRoutingKeyV1(input)
	default:
		return RoutingKey{}, fmt.Errorf("unsupported routing key version: %s", version)
	}
}

func BuildRoutingKeyV1(input SubAccountRouteInput) (RoutingKey, error) {
	if err := input.Validate(); err != nil {
		return RoutingKey{}, err
	}

	value := strings.Join([]string{
		"currency:" + input.CurrencyDimensionID,
		"tax_code:" + routeDimensionValue(input.TaxCodeDimensionID),
		"features:" + routeDimensionValue(input.FeaturesDimensionID),
		"credit_priority:" + routeDimensionValue(input.CreditPriorityDimensionID),
	}, "|")

	return NewRoutingKey(RoutingKeyVersionV1, value)
}

func routeDimensionValue(id *string) string {
	if id == nil || *id == "" {
		return "null"
	}
	return *id
}
