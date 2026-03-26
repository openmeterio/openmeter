package reconciler

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type SemanticPatchOperation string

const (
	SemanticPatchOperationCreate  SemanticPatchOperation = "create"
	SemanticPatchOperationDelete  SemanticPatchOperation = "delete"
	SemanticPatchOperationShrink  SemanticPatchOperation = "shrink"
	SemanticPatchOperationExtend  SemanticPatchOperation = "extend"
	SemanticPatchOperationProrate SemanticPatchOperation = "prorate"
)

type ExpandInput struct {
	Subscription subscription.SubscriptionView
	Currency     currencyx.Calculator
	Invoices     persistedstate.Invoices
}

type SemanticPatch interface {
	semanticPatch()
	Operation() SemanticPatchOperation
	UniqueReferenceID() string
	Expand(ctx context.Context, input ExpandInput) ([]Patch, error)
}
