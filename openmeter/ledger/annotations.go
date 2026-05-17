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
	AnnotationCollectionType          = "ledger.collection.type"
	AnnotationCollectionSourceOrder   = "ledger.collection.source_order"
	AnnotationBreakageKind            = "ledger.breakage.kind"
	AnnotationBreakageRecordID        = "ledger.breakage.record_id"
	AnnotationBreakagePlanID          = "ledger.breakage.plan_id"
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

const CollectionTypeBreakage = "breakage"

type BreakageKind string

const (
	BreakageKindPlan    BreakageKind = "plan"
	BreakageKindRelease BreakageKind = "release"
	BreakageKindReopen  BreakageKind = "reopen"
)

func (BreakageKind) Values() []string {
	return []string{
		string(BreakageKindPlan),
		string(BreakageKindRelease),
		string(BreakageKindReopen),
	}
}

type BreakageSourceKind string

const (
	BreakageSourceKindCreditPurchase           BreakageSourceKind = "credit_purchase"
	BreakageSourceKindUsage                    BreakageSourceKind = "usage"
	BreakageSourceKindUsageCorrection          BreakageSourceKind = "usage_correction"
	BreakageSourceKindCreditPurchaseCorrection BreakageSourceKind = "credit_purchase_correction"
	BreakageSourceKindAdvanceBackfill          BreakageSourceKind = "advance_backfill"
)

func (BreakageSourceKind) Values() []string {
	return []string{
		string(BreakageSourceKindCreditPurchase),
		string(BreakageSourceKindUsage),
		string(BreakageSourceKindUsageCorrection),
		string(BreakageSourceKindCreditPurchaseCorrection),
		string(BreakageSourceKindAdvanceBackfill),
	}
}

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

func BreakageAnnotations(kind BreakageKind, recordID string, planID *string) models.Annotations {
	annotations := models.Annotations{
		AnnotationCollectionType:   CollectionTypeBreakage,
		AnnotationBreakageKind:     string(kind),
		AnnotationBreakageRecordID: recordID,
	}

	if planID != nil && *planID != "" {
		annotations[AnnotationBreakagePlanID] = *planID
	}

	return annotations
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
