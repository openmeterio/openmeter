package balanceworker

import "github.com/openmeterio/openmeter/internal/entitlement/balanceworker"

type (
	Worker           = balanceworker.Worker
	WorkerOptions    = balanceworker.WorkerOptions
	WorkerDLQOptions = balanceworker.WorkerDLQOptions
)

func New(opts WorkerOptions) (*Worker, error) {
	return balanceworker.New(opts)
}
