package service

import "github.com/openmeterio/openmeter/openmeter/taxcode"

var _ taxcode.DeleteValidatorService = (*Service)(nil)

func (s *Service) RegisterDeleteValidator(v taxcode.DeleteValidator) {
	s.deleteValidators.Register(v)
}
