package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type NextSequenceNumberInput struct {
	Namespace string
	Scope     string
}

func (n NextSequenceNumberInput) Validate() error {
	if n.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if n.Scope == "" {
		return fmt.Errorf("scope is required")
	}

	return nil
}

type SequenceDefinition struct {
	Template string
	Scope    string
}

func (d SequenceDefinition) Validate() error {
	if d.Template == "" {
		return fmt.Errorf("prefix is required")
	}

	if d.Scope == "" {
		return fmt.Errorf("scope is required")
	}

	return nil
}

var (
	GatheringInvoiceSequenceNumber = SequenceDefinition{
		Template: "GATHER-{{.CustomerPrefix}}-{{.Currency}}-{{.NextSequenceNumber}}",
		Scope:    "invoices/gathering",
	}
	DraftInvoiceSequenceNumber = SequenceDefinition{
		Template: "DRAFT-{{.CustomerPrefix}}-{{.NextSequenceNumber}}",
		Scope:    "invoices/draft",
	}
)

type SequenceGenerationInput struct {
	Namespace    string
	CustomerName string
	Currency     currencyx.Code
}

func (i SequenceGenerationInput) Validate() error {
	if i.CustomerName == "" {
		return fmt.Errorf("customer name is required")
	}

	if i.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if i.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	return nil
}
