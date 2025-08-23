package clickhouse

import (
	"bytes"
	"fmt"

	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// customerIdSelect returns the select columns for the customer ID.
func selectCustomerIdColumns(eventsTableName string, customers []streaming.Customer) string {
	getColumn := columnFactory(eventsTableName)
	subjectColumn := getColumn("subject")
	added := false

	var caseBuilder bytes.Buffer
	caseBuilder.WriteString("CASE ")

	// Add the case statements for each subject to customer ID mapping
	for _, customer := range customers {
		for _, subjectKey := range customer.GetUsageAttribution().SubjectKeys {
			str := fmt.Sprintf(
				"WHEN %s = '%s' THEN '%s' ",
				subjectColumn,
				sqlbuilder.Escape(subjectKey),
				sqlbuilder.Escape(customer.GetUsageAttribution().ID),
			)
			caseBuilder.WriteString(str)
			added = true
		}
	}

	if !added {
		// No mappings: return a constant column to avoid invalid CASE
		return "'' AS customer_id"
	}

	caseBuilder.WriteString("ELSE '' END AS customer_id")
	return caseBuilder.String()
}

// customersWhereExpr returns the WHERE expression for the customer filter.
func customersWhereExpr(eventsTableName string, customers []streaming.Customer, query *sqlbuilder.SelectBuilder) string {
	// Helper function to filter by subject
	getColumn := columnFactory(eventsTableName)
	subjectColumn := getColumn("subject")

	// If the customer filter is provided, we add all the subjects to the filter
	if len(customers) > 0 {
		subjects := lo.Map(customers, func(customer streaming.Customer, _ int) []string {
			return customer.GetUsageAttribution().SubjectKeys
		})

		return query.In(subjectColumn, lo.Flatten(subjects))
	}

	return ""
}

// subjectWhere applies the subject filter to the query.
// This is a helper function to filter by customers or subjects in the row event table.
func subjectWhere(
	eventsTableName string,
	customers []streaming.Customer,
	subjects []string,
	query *sqlbuilder.SelectBuilder,
) *sqlbuilder.SelectBuilder {
	// Helper function to filter by subject
	getColumn := columnFactory(eventsTableName)
	subjectColumn := getColumn("subject")

	// If the customer filter is provided, we add all the subjects to the filter
	if len(customers) > 0 {
		expr := customersWhereExpr(eventsTableName, customers, query)
		if expr != "" {
			query.Where(expr)
		}
	}

	// If we have a subject filter, we add it to the query
	// If we have both a customer filter and a subject filter,
	// this is an AND between the two filters
	if len(subjects) > 0 {
		query = query.Where(query.In(subjectColumn, subjects))
	}

	return query
}
