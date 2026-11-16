package adapter

import (
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// TODO(notification-v4): retire mixed-storage predicates once all writers and rows
// use versioned names. V1/V3 still need the public name mapping after cutover.
var eventTypeNames = [...]struct {
	legacy    notification.EventType
	versioned notification.EventType
}{
	{notification.EventTypeBalanceThreshold, "v1.entitlement.balance_threshold.triggered"},
	{notification.EventTypeEntitlementReset, "v1.entitlement.usage_period.reset"},
	{notification.EventTypeInvoiceCreated, "v1.billing.invoice.created"},
	{notification.EventTypeInvoiceUpdated, "v1.billing.invoice.updated"},
}

func fromDBEventType(value notification.EventType) notification.EventType {
	for _, names := range eventTypeNames {
		if value == names.versioned {
			return names.legacy
		}
	}

	return value
}

func storedEventTypeNames(values ...string) []string {
	names := make([]string, 0, 2*len(values))
	for _, value := range values {
		names = append(names, value)

		for _, pair := range eventTypeNames {
			if value == string(pair.legacy) {
				names = append(names, string(pair.versioned))
				break
			}
		}
	}

	return names
}

func eventTypePredicate(field string, f *filter.FilterString) func(*sql.Selector) {
	switch {
	case f.Eq != nil:
		return sql.FieldIn(field, storedEventTypeNames(*f.Eq)...)
	case f.Ne != nil:
		return sql.FieldNotIn(field, storedEventTypeNames(*f.Ne)...)
	case f.In != nil:
		return sql.FieldIn(field, storedEventTypeNames((*f.In)...)...)
	case f.Nin != nil:
		return sql.FieldNotIn(field, storedEventTypeNames((*f.Nin)...)...)
	case f.Exists != nil:
		return f.Select(field)
	default:
		// Internal callers can also supply pattern, range, or compound filters.
		// Let PostgreSQL apply those to the same name that callers see on reads.
		return func(s *sql.Selector) {
			d := sql.Dialect(s.Dialect())
			names := d.Select().AppendSelectExprAs(legacyEventTypeExpr(s.C(field)), field).As("event_type_names")
			matches := d.Select().From(names)
			f.Select(names.C(field))(matches)

			s.Where(sql.Exists(matches))
		}
	}
}

func legacyEventTypeExpr(column string) sql.Querier {
	return sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("CASE ").Ident(column)

		for _, names := range eventTypeNames {
			b.WriteString(" WHEN ").Arg(names.versioned).WriteString(" THEN ").Arg(names.legacy)
		}

		b.WriteString(" ELSE ").Ident(column).WriteString(" END")
	})
}

func byLegacyEventType(field string, order sortx.Order) func(*sql.Selector) {
	return func(s *sql.Selector) {
		name := legacyEventTypeExpr(s.C(field))
		if order == sortx.OrderDesc {
			name = sql.DescExpr(name)
		}

		s.OrderExpr(name)
	}
}
