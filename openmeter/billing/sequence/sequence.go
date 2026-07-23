package sequence

import (
	"fmt"
	"slices"
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

type Definition struct {
	Prefix         string
	SuffixTemplate string
	Scope          string
	CommitMode     CommitMode
}

func (d Definition) Validate() error {
	if d.Prefix == "" {
		return fmt.Errorf("prefix is required")
	}

	if d.SuffixTemplate == "" {
		return fmt.Errorf("suffix template is required")
	}

	if d.Scope == "" {
		return fmt.Errorf("scope is required")
	}

	if err := d.CommitMode.Validate(); err != nil {
		return err
	}

	return nil
}

func (d Definition) PrefixMatches(s string) bool {
	return strings.HasPrefix(s, d.Prefix+"-")
}

// CommitMode controls when a sequence allocation is committed. WithCaller
// allows the number to be reused if the caller rolls back; Independent retains
// the allocation despite caller rollback, which can create gaps.
type CommitMode string

const (
	CommitModeWithCaller  CommitMode = "with_caller"
	CommitModeIndependent CommitMode = "independent"
)

func (m CommitMode) Validate() error {
	if m == "" {
		return fmt.Errorf("commit mode is required")
	}

	if !slices.Contains([]CommitMode{CommitModeWithCaller, CommitModeIndependent}, m) {
		return fmt.Errorf("commit mode is invalid: %s", m)
	}

	return nil
}

var (
	GatheringInvoiceSequenceNumber = Definition{
		Prefix:         "GATHER",
		SuffixTemplate: "{{.CustomerPrefix}}-{{.Currency}}-{{.NextSequenceNumber}}",
		Scope:          "invoices/gathering",
		CommitMode:     CommitModeIndependent,
	}
	DraftInvoiceSequenceNumber = Definition{
		Prefix:         "DRAFT",
		SuffixTemplate: "{{.CustomerPrefix}}-{{.NextSequenceNumber}}",
		Scope:          "invoices/draft",
		// Draft numbers are temporary and replaced by the invoicing app's final
		// invoice number, so gaps from retained allocations are acceptable.
		CommitMode: CommitModeIndependent,
	}
)

type GenerationInput struct {
	Namespace    string
	CustomerName string
	Currency     currencyx.FiatCode
}

func (i GenerationInput) Validate() error {
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
