package service

import "github.com/openmeterio/openmeter/pkg/framework/lockr"

func NewEntitlementUniqueScopeLock(featureKey string, subjectKey string) (lockr.Key, error) {
	return lockr.NewKey("fk", featureKey, "sk", subjectKey)
}
