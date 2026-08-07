package detailedline

import "github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"

type Creator[T any] interface {
	stddetailedline.Creator[T]

	SetAmountDiscounts(AmountDiscounts) T
}

func Create[T Creator[T]](creator T, line Base) T {
	create := stddetailedline.Create(creator, line.Base)

	return create.SetAmountDiscounts(line.AmountDiscounts)
}

type DBGetter interface {
	stddetailedline.DBGetter

	GetAmountDiscounts() AmountDiscounts
}

func FromDB[T DBGetter](dbEntity T) Base {
	return Base{
		Base:            stddetailedline.FromDB(dbEntity),
		AmountDiscounts: dbEntity.GetAmountDiscounts(),
	}
}
