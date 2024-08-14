package balanceworker

import "github.com/openmeterio/openmeter/internal/entitlement/balanceworker"

type (
	Worker        = balanceworker.Worker
	WorkerOptions = balanceworker.WorkerOptions

	SubjectResolver = balanceworker.SubjectResolver
)

func New(opts WorkerOptions) (*Worker, error) {
	return balanceworker.New(opts)
}

type (
	Recalculator        = balanceworker.Recalculator
	RecalculatorOptions = balanceworker.RecalculatorOptions
)

const (
	DefaultIncludeDeletedDuration = balanceworker.DefaultIncludeDeletedDuration
)

func NewRecalculator(opts RecalculatorOptions) (*Recalculator, error) {
	return balanceworker.NewRecalculator(opts)
}
