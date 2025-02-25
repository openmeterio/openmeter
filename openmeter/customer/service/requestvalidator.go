package customerservice

import "github.com/openmeterio/openmeter/openmeter/customer"

var _ customer.RequestValidatorService = (*Service)(nil)

func (s *Service) RegisterRequestValidator(v customer.RequestValidator) {
	s.requestValidatorRegistry.Register(v)
}
