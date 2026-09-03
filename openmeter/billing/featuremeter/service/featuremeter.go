package featuremeterservice

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type FeatureMeterCollection struct {
	ByKey map[string]billingfeaturemeter.FeatureMeter
	ByID  map[string]billingfeaturemeter.FeatureMeter
}

func (f FeatureMeterCollection) Get(reference billingfeaturemeter.FeatureReferenceGetter) (billingfeaturemeter.FeatureMeter, error) {
	if reference == nil {
		return billingfeaturemeter.FeatureMeter{}, models.NewGenericValidationError(errors.New("feature reference is required"))
	}

	featureRef := reference.GetFeatureMeterRef()
	if featureRef == nil || lo.IsEmpty(featureRef.IDOrKey) {
		return billingfeaturemeter.FeatureMeter{}, models.NewGenericValidationError(errors.New("feature reference is required"))
	}

	var featureMeter billingfeaturemeter.FeatureMeter
	var exists bool
	var referenceValue string
	if featureRef.IDOrKey.ID != "" {
		featureMeter, exists = f.ByID[featureRef.IDOrKey.ID]
		referenceValue = featureRef.IDOrKey.ID
	} else {
		featureMeter, exists = f.ByKey[featureRef.IDOrKey.Key]
		referenceValue = featureRef.IDOrKey.Key
	}
	if !exists {
		return billingfeaturemeter.FeatureMeter{}, newValidationIssueWithIdentity(
			reference,
			billing.ValidationWithMessagef(
				billing.ErrInvoiceLineFeatureNotFound,
				"feature[%s]",
				referenceValue,
			),
		)
	}

	if featureRef.RequireMeter && featureMeter.Meter == nil {
		return featureMeter, newValidationIssueWithIdentity(
			reference,
			billing.ValidationWithMessagef(
				billing.ErrInvoiceLineFeatureHasNoMeters,
				"feature[%s]",
				featureMeter.Feature.Key,
			),
		)
	}

	return featureMeter, nil
}

func (f FeatureMeterCollection) Has(reference billingfeaturemeter.FeatureReferenceGetter) bool {
	if reference == nil {
		return false
	}

	featureRef := reference.GetFeatureMeterRef()
	if featureRef == nil {
		return false
	}

	if featureRef.IDOrKey.ID != "" {
		_, exists := f.ByID[featureRef.IDOrKey.ID]

		return exists
	}

	if featureRef.IDOrKey.Key != "" {
		_, exists := f.ByKey[featureRef.IDOrKey.Key]

		return exists
	}

	return false
}

func newValidationIssueWithIdentity(reference billingfeaturemeter.FeatureReferenceGetter, err error) error {
	owner, ok := reference.(billingfeaturemeter.FeatureReferenceOwner)
	if !ok {
		return err
	}

	identity := owner.GetFeatureMeterOwner()
	if identity.Kind == "" || identity.ID == "" {
		return err
	}

	return billing.ValidationWithFieldPrefix(fmt.Sprintf("%s/%s", identity.Kind, identity.ID), err)
}
