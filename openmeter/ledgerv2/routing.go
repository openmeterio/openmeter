package ledgerv2

import "fmt"

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
