package subscription

import "github.com/openmeterio/openmeter/pkg/models"

// MapSubscriptionSpecValidationIssueFieldSelectors maps the FieldSelectors of a ValidationIssue from the structure of SubscriptionSpec
// to the structure of api.SubscriptionView
func MapSubscriptionSpecValidationIssueFieldSelectors(iss models.ValidationIssue) (models.ValidationIssue, error) {
	currFields := iss.Field()

	if len(currFields) < 2 || currFields[0].String() != "phases" {
		return iss, nil
	}

	phaseKey := currFields[1].String()
	newFields := models.NewFieldSelectors(
		append(
			models.FieldSelectors{
				currFields[0].WithExpression(models.NewFieldAttrValue("key", phaseKey)),
			},
			currFields[2:]...,
		)...,
	)

	iss = iss.WithField(newFields...)

	return iss, nil
}
