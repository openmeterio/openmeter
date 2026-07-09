package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template" // nosemgrep

	"github.com/gosimple/unidecode"

	"github.com/openmeterio/openmeter/openmeter/billing/sequence"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type Service struct {
	adapter sequence.Adapter
}

var _ sequence.Service = (*Service)(nil)

func New(adapter sequence.Adapter) *Service {
	return &Service{
		adapter: adapter,
	}
}

type sequenceInput struct {
	CustomerPrefix     string
	Currency           currencyx.Code
	NextSequenceNumber string
}

func (s *Service) GenerateInvoiceSequenceNumber(ctx context.Context, in sequence.GenerationInput, def sequence.Definition) (string, error) {
	if err := in.Validate(); err != nil {
		return "", err
	}

	if err := def.Validate(); err != nil {
		return "", err
	}

	nextSequenceNumber, err := s.adapter.NextSequenceNumber(ctx, sequence.NextSequenceNumberInput{
		Namespace: in.Namespace,
		Scope:     def.Scope,
	})
	if err != nil {
		return "", err
	}

	input := sequenceInput{
		CustomerPrefix:     getCustomerPrefix(in.CustomerName),
		Currency:           in.Currency,
		NextSequenceNumber: nextSequenceNumber.String(),
	}

	tmpl, err := template.New("invoiceseq").Parse(def.SuffixTemplate)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer

	if err := tmpl.Execute(&out, input); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", def.Prefix, out.String()), nil
}

func getCustomerPrefix(name string) string {
	asciiName := unidecode.Unidecode(name)

	components := strings.Split(strings.ToUpper(asciiName), " ")
	if len(components) == 0 || (len(components) == 1 && components[0] == "") {
		return "UNKN"
	}

	if len(components) == 1 {
		return safeSubStr(components[0], 4)
	}

	return safeSubStr(components[0], 2) + safeSubStr(components[1], 2)
}

func safeSubStr(str string, length int) string {
	if len(str) <= length {
		return str
	}

	return str[0:length]
}
