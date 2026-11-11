package rte

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
)

type TargetIdentity struct {
	Namespace string
	OwnerID   string
	KeyHash   string
}

// Identity follows stored-state identity, including the distinction between all,
// unrestricted and feature-scoped credit balances. JSON spelling is irrelevant.
func (t CalculationTarget) Identity() (TargetIdentity, error) {
	key, err := t.DecodeKey()
	if err != nil {
		return TargetIdentity{}, err
	}

	var identity TargetIdentity
	var dimensions []string
	switch key := key.(type) {
	case creditbalance.Key:
		identity.Namespace, identity.OwnerID = key.CustomerID.Namespace, key.CustomerID.ID
		dimensions = []string{string(t.Kind), identity.Namespace, identity.OwnerID, key.Currency.IdentityKey(), key.FeatureScope()}
	case meteredentitlement.Key:
		identity.Namespace, identity.OwnerID = key.EntitlementID.Namespace, key.EntitlementID.ID
		dimensions = []string{string(t.Kind), identity.Namespace, identity.OwnerID}
	}

	encoded, err := json.Marshal(dimensions)
	if err != nil {
		return TargetIdentity{}, err
	}

	hash := sha256.Sum256(encoded)
	identity.KeyHash = hex.EncodeToString(hash[:])

	return identity, nil
}
