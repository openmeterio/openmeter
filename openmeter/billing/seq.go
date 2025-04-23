package billing

import (
	"fmt"
	"strings"

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
	Prefix         string
	SuffixTemplate string
	Scope          string
}

func (d SequenceDefinition) Validate() error {
	if d.Prefix == "" {
		return fmt.Errorf("prefix is required")
	}

	if d.SuffixTemplate == "" {
		return fmt.Errorf("suffix template is required")
	}

	if d.Scope == "" {
		return fmt.Errorf("scope is required")
	}

	return nil
}

func (d SequenceDefinition) PrefixMatches(s string) bool {
	return strings.HasPrefix(s, d.Prefix+"-")
}

var (
	GatheringInvoiceSequenceNumber = SequenceDefinition{
		Prefix:         "GATHER",
		SuffixTemplate: "{{.CustomerPrefix}}-{{.Currency}}-{{.NextSequenceNumber}}",
		Scope:          "invoices/gathering",
	}
	DraftInvoiceSequenceNumber = SequenceDefinition{
		Prefix:         "DRAFT",
		SuffixTemplate: "{{.CustomerPrefix}}-{{.NextSequenceNumber}}",
		Scope:          "invoices/draft",
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
