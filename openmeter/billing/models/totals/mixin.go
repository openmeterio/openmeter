package totals

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"
)

type Mixin struct {
	mixin.Schema
}

func (m Mixin) Fields() []ent.Field {
	return []ent.Field{
		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("taxes_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("taxes_inclusive_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("taxes_exclusive_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("charges_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("discounts_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("credits_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

type Mutator[T any] interface {
	SetAmount(amount alpacadecimal.Decimal) T
	SetTaxesTotal(taxesTotal alpacadecimal.Decimal) T
	SetTaxesInclusiveTotal(taxesInclusiveTotal alpacadecimal.Decimal) T
	SetTaxesExclusiveTotal(taxesExclusiveTotal alpacadecimal.Decimal) T
	SetChargesTotal(chargesTotal alpacadecimal.Decimal) T
	SetDiscountsTotal(discountsTotal alpacadecimal.Decimal) T
	SetCreditsTotal(creditsTotal alpacadecimal.Decimal) T
	SetTotal(total alpacadecimal.Decimal) T
}

func Set[T Mutator[T]](mut Mutator[T], totals Totals) T {
	return mut.SetAmount(totals.Amount).
		SetTaxesTotal(totals.TaxesTotal).
		SetTaxesInclusiveTotal(totals.TaxesInclusiveTotal).
		SetTaxesExclusiveTotal(totals.TaxesExclusiveTotal).
		SetChargesTotal(totals.ChargesTotal).
		SetDiscountsTotal(totals.DiscountsTotal).
		SetCreditsTotal(totals.CreditsTotal).
		SetTotal(totals.Total)
}

type TotalsGetter interface {
	GetAmount() alpacadecimal.Decimal
	GetTaxesTotal() alpacadecimal.Decimal
	GetTaxesInclusiveTotal() alpacadecimal.Decimal
	GetTaxesExclusiveTotal() alpacadecimal.Decimal
	GetChargesTotal() alpacadecimal.Decimal
	GetDiscountsTotal() alpacadecimal.Decimal
	GetCreditsTotal() alpacadecimal.Decimal
	GetTotal() alpacadecimal.Decimal
}

func FromDB(e TotalsGetter) Totals {
	return Totals{
		Amount:              e.GetAmount(),
		TaxesTotal:          e.GetTaxesTotal(),
		TaxesInclusiveTotal: e.GetTaxesInclusiveTotal(),
		TaxesExclusiveTotal: e.GetTaxesExclusiveTotal(),
		ChargesTotal:        e.GetChargesTotal(),
		DiscountsTotal:      e.GetDiscountsTotal(),
		CreditsTotal:        e.GetCreditsTotal(),
		Total:               e.GetTotal(),
	}
}
