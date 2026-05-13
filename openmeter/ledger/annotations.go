package ledger

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	AnnotationChargeNamespace     = "ledger.charge.namespace"
	AnnotationChargeID            = "ledger.charge.id"
	AnnotationSubscriptionID      = "ledger.subscription.id"
	AnnotationSubscriptionPhaseID = "ledger.subscription.phase.id"
	AnnotationSubscriptionItemID  = "ledger.subscription.item.id"
	AnnotationFeatureID           = "ledger.feature.id"

	AnnotationTransactionTemplateCode = "ledger.transaction.template_code"
	AnnotationTransactionDirection    = "ledger.transaction.direction"
)

type ChargeTransactionAnnotationsInput struct {
	ChargeID models.NamespacedID

	SubscriptionID      *string
	SubscriptionPhaseID *string
	SubscriptionItemID  *string
	FeatureID           *string
}

type TransactionDirection string

const (
	TransactionDirectionForward    TransactionDirection = "forward"
	TransactionDirectionCorrection TransactionDirection = "correction"
)

func ChargeAnnotations(chargeID models.NamespacedID) models.Annotations {
	return models.Annotations{
		AnnotationChargeNamespace: chargeID.Namespace,
		AnnotationChargeID:        chargeID.ID,
	}
}

func ChargeTransactionAnnotations(input ChargeTransactionAnnotationsInput) models.Annotations {
	annotations := ChargeAnnotations(input.ChargeID)

	if input.SubscriptionID != nil && *input.SubscriptionID != "" {
		annotations[AnnotationSubscriptionID] = *input.SubscriptionID
	}

	if input.SubscriptionPhaseID != nil && *input.SubscriptionPhaseID != "" {
		annotations[AnnotationSubscriptionPhaseID] = *input.SubscriptionPhaseID
	}

	if input.SubscriptionItemID != nil && *input.SubscriptionItemID != "" {
		annotations[AnnotationSubscriptionItemID] = *input.SubscriptionItemID
	}

	if input.FeatureID != nil && *input.FeatureID != "" {
		annotations[AnnotationFeatureID] = *input.FeatureID
	}

	return annotations
}

func TransactionAnnotations(templateCode string, direction TransactionDirection) models.Annotations {
	return models.Annotations{
		AnnotationTransactionTemplateCode: templateCode,
		AnnotationTransactionDirection:    string(direction),
	}
}

func TransactionTemplateCodeFromAnnotations(annotations models.Annotations) (string, error) {
	raw, ok := annotations[AnnotationTransactionTemplateCode]
	if !ok {
		return "", fmt.Errorf("transaction template code annotation is required")
	}

	code, ok := raw.(string)
	if !ok || code == "" {
		return "", fmt.Errorf("transaction template code annotation is invalid")
	}

	return code, nil
}

func TransactionDirectionFromAnnotations(annotations models.Annotations) (TransactionDirection, error) {
	raw, ok := annotations[AnnotationTransactionDirection]
	if !ok {
		return "", fmt.Errorf("transaction direction annotation is required")
	}

	value, ok := raw.(string)
	if !ok || value == "" {
		return "", fmt.Errorf("transaction direction annotation is invalid")
	}

	direction := TransactionDirection(value)
	switch direction {
	case TransactionDirectionForward, TransactionDirectionCorrection:
		return direction, nil
	default:
		return "", fmt.Errorf("invalid transaction direction annotation %q", value)
	}
}
