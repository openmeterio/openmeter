package rte

import "context"

type Worker interface {
	Run(context.Context) error
	Close() error
	Running() <-chan struct{}
}
