package clickhouse

import (
	"fmt"
	"strings"

	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// selectCustomerIdColumn
func selectCustomerIdColumn(eventsTableName string, customers []streaming.Customer, query *sqlbuilder.SelectBuilder) *sqlbuilder.SelectBuilder {
	// If there are no customers, we return an empty customer id column
	if len(customers) == 0 {
		return query.SelectMore("'' AS customer_id")
	}

	// Helper function to get the subject column
	getColumn := columnFactory(eventsTableName)
	subjectColumn := getColumn("subject")

	// Build a map of subject to customer id
	var values []string

	for _, customer := range customers {
		// Add each subject key to the map and map it to the customer id
		for _, subjectKey := range customer.GetUsageAttribution().SubjectKeys {
			subjectSQL := fmt.Sprintf("'%s'", sqlbuilder.Escape(subjectKey))
			customerIDSQL := fmt.Sprintf("'%s'", sqlbuilder.Escape(customer.GetUsageAttribution().ID))

			values = append(values, subjectSQL, customerIDSQL)
		}
	}

	mapAs := "subject_to_customer_id"
	mapSQL := fmt.Sprintf("WITH map(%s) as %s", strings.Join(values, ", "), mapAs)

	// Add the map to query via WITH clause
	mapQuery := sqlbuilder.ClickHouse.NewCTEBuilder().SQL(mapSQL)
	query = query.With(mapQuery)

	// Select the customer id column
	query = query.SelectMore(fmt.Sprintf("%s[%s] AS customer_id", mapAs, subjectColumn))

	return query
}

// customersWhere applies the customer filter to the query.
func customersWhere(eventsTableName string, customers []streaming.Customer, query *sqlbuilder.SelectBuilder) *sqlbuilder.SelectBuilder {
	// If there are no customers, we return an empty subject filter
	if len(customers) == 0 {
		return query
	}

	// Helper function to filter by subject
	getColumn := columnFactory(eventsTableName)
	subjectColumn := getColumn("subject")

	// If the customer filter is provided, we add all the subjects to the filter
	subjects := lo.Map(customers, func(customer streaming.Customer, _ int) []string {
		return customer.GetUsageAttribution().SubjectKeys
	})

	return query.Where(query.In(subjectColumn, lo.Flatten(subjects)))
}

// subjectWhere applies the subject filter to the query.
func subjectWhere(
	eventsTableName string,
	subjects []string,
	query *sqlbuilder.SelectBuilder,
) *sqlbuilder.SelectBuilder {
	// Helper function to filter by subject
	getColumn := columnFactory(eventsTableName)
	subjectColumn := getColumn("subject")

	// If we have a subject filter, we add it to the query
	// If we have both a customer filter and a subject filter,
	// this is an AND between the two filters
	if len(subjects) > 0 {
		query = query.Where(query.In(subjectColumn, subjects))
	}

	return query
}

func columnFactory(alias string) func(string) string {
	return func(column string) string {
		return fmt.Sprintf("%s.%s", alias, column)
	}
}
