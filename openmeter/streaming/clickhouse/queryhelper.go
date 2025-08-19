package clickhouse

import (
	"bytes"
	"fmt"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// customerIdSelect returns the select columns for the customer ID.
func selectCustomerIdColumns(eventsTableName string, customers []streaming.Customer) []string {
	getColumn := columnFactory(eventsTableName)
	subjectColumn := getColumn("subject")

	selectColumns := []string{}
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
		}
	}

	// If the subject is not in the map, we return an empty string
	caseBuilder.WriteString("ELSE '' END AS customer_id")

	// Add the case statement to the select columns
	selectColumns = append(selectColumns, caseBuilder.String())

	return selectColumns
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

	mapFunc := func(subject string) string {
		return query.Equal(subjectColumn, subject)
	}

	// If the customer filter is provided, we add all the subjects to the filter
	if len(customers) > 0 {
		var subjects []string

		for _, customer := range customers {
			subjects = append(subjects, customer.GetUsageAttribution().SubjectKeys...)
		}

		query = query.Where(query.Or(slicesx.Map(subjects, mapFunc)...))
	}

	// If we have a subject filter, we add it to the query
	// If we have both a customer filter and a subject filter,
	// this is an AND between the two filters
	if len(subjects) > 0 {
		query = query.Where(query.Or(slicesx.Map(subjects, mapFunc)...))
	}

	return query
}
