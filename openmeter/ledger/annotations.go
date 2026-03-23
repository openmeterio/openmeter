package ledger

import "github.com/openmeterio/openmeter/pkg/models"

const (
	AnnotationChargeNamespace = "ledger.charge.namespace"
	AnnotationChargeID        = "ledger.charge.id"
)

func ChargeAnnotations(chargeID models.NamespacedID) models.Annotations {
	return models.Annotations{
		AnnotationChargeNamespace: chargeID.Namespace,
		AnnotationChargeID:        chargeID.ID,
	}
}
