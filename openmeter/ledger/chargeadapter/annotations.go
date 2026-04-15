package chargeadapter

import (
	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargeflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargeusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

func chargeAnnotationsForCreditPurchaseCharge(charge chargecreditpurchase.Charge) models.Annotations {
	return chargeTransactionAnnotations(
		models.NamespacedID{
			Namespace: charge.Namespace,
			ID:        charge.ID,
		},
		charge.Intent.Subscription,
		nil,
	)
}

func chargeAnnotationsForFlatFeeCharge(charge chargeflatfee.Charge) models.Annotations {
	return chargeTransactionAnnotations(
		models.NamespacedID{
			Namespace: charge.Namespace,
			ID:        charge.ID,
		},
		charge.Intent.Subscription,
		charge.State.FeatureID,
	)
}

func chargeAnnotationsForUsageBasedCharge(charge chargeusagebased.Charge) models.Annotations {
	return chargeTransactionAnnotations(
		models.NamespacedID{
			Namespace: charge.Namespace,
			ID:        charge.ID,
		},
		charge.Intent.Subscription,
		ptrIfNotEmpty(charge.State.FeatureID),
	)
}

func chargeTransactionAnnotations(chargeID models.NamespacedID, subscription *meta.SubscriptionReference, featureID *string) models.Annotations {
	var subscriptionID *string
	var subscriptionPhaseID *string
	var subscriptionItemID *string

	if subscription != nil {
		subscriptionID = &subscription.SubscriptionID
		subscriptionPhaseID = &subscription.PhaseID
		subscriptionItemID = &subscription.ItemID
	}

	return ledger.ChargeTransactionAnnotations(ledger.ChargeTransactionAnnotationsInput{
		ChargeID:            chargeID,
		SubscriptionID:      subscriptionID,
		SubscriptionPhaseID: subscriptionPhaseID,
		SubscriptionItemID:  subscriptionItemID,
		FeatureID:           featureID,
	})
}

func ptrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
