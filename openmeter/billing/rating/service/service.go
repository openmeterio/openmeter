package service

import "github.com/openmeterio/openmeter/openmeter/billing/rating"

type service struct{}

func New() rating.Service {
	return &service{}
}
