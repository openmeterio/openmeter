package service

import "github.com/openmeterio/openmeter/openmeter/billing/pricer"

type service struct{}

func New() pricer.Service {
	return &service{}
}
