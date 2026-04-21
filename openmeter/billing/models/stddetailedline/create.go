package stddetailedline

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	billingtotals "github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type Creator[T any] interface {
	externalid.LineExternalIDCreator[T]
	billingtotals.Setter[T]

	SetName(string) T
	SetNillableDescription(*string) T
	SetCurrency(currencyx.Code) T
	SetServicePeriodStart(time.Time) T
	SetServicePeriodEnd(time.Time) T
	SetQuantity(alpacadecimal.Decimal) T
	SetPerUnitAmount(alpacadecimal.Decimal) T
	SetCategory(Category) T
	SetPaymentTerm(productcatalog.PaymentTermType) T
	SetNillableIndex(*int) T
	SetChildUniqueReferenceID(string) T
	SetNillableDeletedAt(*time.Time) T
}

func Create[T Creator[T]](creator Creator[T], line Base) T {
	create := creator.
		SetName(line.Name).
		SetNillableDescription(line.Description).
		SetCurrency(line.Currency).
		SetServicePeriodStart(line.ServicePeriod.From.In(time.UTC)).
		SetServicePeriodEnd(line.ServicePeriod.To.In(time.UTC)).
		SetQuantity(line.Quantity).
		SetPerUnitAmount(line.PerUnitAmount).
		SetCategory(line.Category).
		SetPaymentTerm(line.PaymentTerm).
		SetNillableIndex(line.Index).
		SetChildUniqueReferenceID(line.ChildUniqueReferenceID).
		SetNillableDeletedAt(line.DeletedAt)

	create = externalid.CreateLineExternalID(create, line.ExternalIDs)
	create = billingtotals.Set(create, line.Totals)

	return create
}
