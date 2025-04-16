package service

import (
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type WorkflowServiceConfig struct {
	Service      subscription.Service
	AddonService subscriptionaddon.Service
	// connectors
	CustomerService customer.Service
	// framework
	TransactionManager transaction.Creator
}

type service struct {
	WorkflowServiceConfig
}

func NewWorkflowService(cfg WorkflowServiceConfig) subscriptionworkflow.Service {
	return &service{
		WorkflowServiceConfig: cfg,
	}
}

var _ subscriptionworkflow.Service = &service{}
