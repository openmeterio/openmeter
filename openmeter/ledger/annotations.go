package ledger

import "github.com/openmeterio/openmeter/pkg/models"

const (
	AnnotationChargeNamespace = "ledger.charge.namespace"
	AnnotationChargeID        = "ledger.charge.id"

	AnnotationTransactionTemplateName = "ledger.transaction.template_name"
	AnnotationTransactionDirection    = "ledger.transaction.direction"
)

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

func TransactionAnnotations(templateName string, direction TransactionDirection) models.Annotations {
	return models.Annotations{
		AnnotationTransactionTemplateName: templateName,
		AnnotationTransactionDirection:    string(direction),
	}
}
