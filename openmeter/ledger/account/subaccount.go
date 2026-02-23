package account

import (
	"time"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubAccountDimensions struct {
	Currency *currencyDimension

	// TODO: implement other dimension types
	TaxCode        mo.Option[ledger.DimensionTaxCode]
	CreditPriority mo.Option[ledger.DimensionCreditPriority]
	Feature        mo.Option[ledger.DimensionFeature]
}

type SubAccountData struct {
	ID          string
	Namespace   string
	Annotations models.Annotations
	CreatedAt   time.Time

	AccountID   string
	AccountType ledger.AccountType

	Dimensions SubAccountDimensions
}

func NewSubAccountFromData(data SubAccountData) (*SubAccount, error) {
	sAcc := &SubAccount{
		data: data,
	}

	return sAcc, nil
}

type SubAccount struct {
	data SubAccountData
}

var _ ledger.SubAccount = (*SubAccount)(nil)

func (s *SubAccount) Address() ledger.PostingAddress {
	return NewAddressFromData(AddressData{
		SubAccountID: s.data.ID,
		AccountType:  s.data.AccountType,
	})
}

func (s *SubAccount) Dimensions() ledger.SubAccountDimensions {
	return ledger.SubAccountDimensions{
		Currency:       s.data.Dimensions.Currency,
		TaxCode:        s.data.Dimensions.TaxCode,
		CreditPriority: s.data.Dimensions.CreditPriority,
		Feature:        s.data.Dimensions.Feature,
	}
}
