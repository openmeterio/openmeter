package clickhouse

import (
	"fmt"

	"github.com/huandu/go-sqlbuilder"

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

	// Build a map of event subjects to customer ids
	var subjectKeys, customerIDs []string

	// For each customer, we map event subjects to customer ids
	for _, customer := range customers {
		usageAttribution := customer.GetUsageAttribution()

		// We map the customer key to the customer id if it exists
		if usageAttribution.Key != nil {
			subjectKeys = append(subjectKeys, *usageAttribution.Key)
			customerIDs = append(customerIDs, usageAttribution.ID)
		}

		// We map each subject key to the customer id
		for _, subjectKey := range usageAttribution.SubjectKeys {
			subjectKeys = append(subjectKeys, subjectKey)
			customerIDs = append(customerIDs, usageAttribution.ID)
		}
	}

	// If there are no values, we return an empty customer id column
	// This can happen if none of the customers has key or usage attribution subjects
	if len(subjectKeys) == 0 {
		return query.SelectMore("'' AS customer_id")
	}

	// Select the customer id column
	query = query.SelectMore(fmt.Sprintf(
		"mapFromArrays(%s, %s)[%s] AS customer_id",
		query.Var(subjectKeys),
		query.Var(customerIDs),
		subjectColumn,
	))

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

	var subjects []string

	// Collect all the subjects from the customers
	for _, customer := range customers {
		subjects = append(subjects, customer.GetUsageAttribution().GetValues()...)
	}

	// If there are no subjects, we return an empty subject filter
	// This can happen if none of the customers has key or usage attribution subjects
	if len(subjects) == 0 {
		return query
	}

	return query.Where(query.In(subjectColumn, subjects))
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
