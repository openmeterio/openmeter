package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/charges"
)

func (s *service) createCreditRealization(ctx context.Context, ch charges.Charge, realizations []charges.CreditRealizationCreateInput) error {
	_, err := s.adapter.CreateCreditRealizations(ctx, ch.GetChargeID(), realizations)
	return err
}
