package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template" // nosemgrep
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/gosimple/unidecode"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/billing/sequence"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Service struct {
	adapter            sequence.Adapter
	allocationDuration metric.Float64Histogram
}

var _ sequence.Service = (*Service)(nil)

type Config struct {
	Adapter sequence.Adapter
	Meter   metric.Meter
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	allocationDuration, err := config.Meter.Float64Histogram(
		"openmeter.billing.sequence.allocation.duration",
		metric.WithDescription("Time spent allocating a billing sequence number"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocation duration histogram: %w", err)
	}

	return &Service{
		adapter:            config.Adapter,
		allocationDuration: allocationDuration,
	}, nil
}

type sequenceInput struct {
	CustomerPrefix     string
	Currency           currencyx.FiatCode
	NextSequenceNumber string
}

func (s *Service) GenerateInvoiceSequenceNumber(ctx context.Context, in sequence.GenerationInput, def sequence.Definition) (string, error) {
	if err := in.Validate(); err != nil {
		return "", err
	}

	if err := def.Validate(); err != nil {
		return "", err
	}

	nextSequenceNumber, err := s.nextSequenceNumber(ctx, def.CommitMode, sequence.NextSequenceNumberInput{
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

func (s *Service) nextSequenceNumber(ctx context.Context, commitMode sequence.CommitMode, input sequence.NextSequenceNumberInput) (alpacadecimal.Decimal, error) {
	startedAt := time.Now()
	defer func() {
		s.allocationDuration.Record(ctx, time.Since(startedAt).Seconds(), metric.WithAttributes(
			attribute.String("scope", input.Scope),
			attribute.String("commit_mode", string(commitMode)),
		))
	}()

	switch commitMode {
	case sequence.CommitModeWithCaller:
		return s.adapter.NextSequenceNumber(ctx, input)
	case sequence.CommitModeIndependent:
		// Avoid holding the namespace-and-scope sequence lock for the caller's full
		// transaction. This favors concurrency over contiguous invoice numbering,
		// so rolled-back callers can leave gaps.
		return transaction.RunInNewTransaction(ctx, s.adapter, func(ctx context.Context) (alpacadecimal.Decimal, error) {
			return s.adapter.NextSequenceNumber(ctx, input)
		})
	default:
		return alpacadecimal.Zero, fmt.Errorf("commit mode is invalid: %s", commitMode)
	}
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
