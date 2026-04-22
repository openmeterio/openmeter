package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.LineEngine = (*LineEngine)(nil)

type LineEngine struct {
	service *service
}

func (e *LineEngine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeFlatFee
}

func (e *LineEngine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	return true, nil
}

func (e *LineEngine) SplitGatheringLine(context.Context, billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{}, fmt.Errorf("flat fee line is not progressively billed")
}

func (e *LineEngine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := slicesx.MapWithErr(input.GatheringLines, func(gatheringLine billing.GatheringLine) (*billing.StandardLine, error) {
		stdLine, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
		if err != nil {
			return nil, fmt.Errorf("converting gathering line to standard line: %w", err)
		}

		return stdLine, nil
	})
	if err != nil {
		return nil, err
	}

	return e.CalculateLines(billing.CalculateLinesInput{
		Invoice: input.Invoice,
		Lines:   stdLines,
	})
}

func (e *LineEngine) OnStandardInvoiceCreated(ctx context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	chargesByID := make(map[string]flatfee.Charge, len(input.Lines))
	for _, stdLine := range input.Lines {
		chargeID := stdLine.ChargeID
		if chargeID == nil {
			return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
		}

		charge, err := e.service.GetByID(ctx, flatfee.GetByIDInput{
			ChargeID: meta.ChargeID{
				Namespace: stdLine.Namespace,
				ID:        *chargeID,
			},
			Expands: meta.Expands{
				meta.ExpandRealizations,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("getting flat fee charge for line[%s]: %w", stdLine.ID, err)
		}
		chargesByID[charge.ID] = charge

		realizations, err := e.service.PostLineAssignedToInvoice(ctx, charge, *stdLine)
		if err != nil {
			return nil, fmt.Errorf("allocating credits for line[%s]: %w", stdLine.ID, err)
		}

		if len(realizations) > 0 {
			stdLine.CreditsApplied = convertCreditRealizations(realizations)
		}
	}

	lines, err := e.CalculateLines(billing.CalculateLinesInput(input))
	if err != nil {
		return nil, err
	}

	for _, stdLine := range lines {
		if stdLine.ChargeID == nil {
			return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
		}

		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return nil, fmt.Errorf("flat fee charge not found for line[%s]: %s", stdLine.ID, *stdLine.ChargeID)
		}

		if err := e.service.persistDetailedLines(ctx, charge, *stdLine); err != nil {
			return nil, fmt.Errorf("persisting detailed lines for line[%s]: %w", stdLine.ID, err)
		}
	}

	return lines, nil
}

func (e *LineEngine) OnCollectionCompleted(_ context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (e *LineEngine) OnInvoiceIssued(_ context.Context, _ billing.OnInvoiceIssuedInput) error {
	return nil
}

func (e *LineEngine) OnPaymentAuthorized(_ context.Context, _ billing.OnPaymentAuthorizedInput) error {
	return nil
}

func (e *LineEngine) OnPaymentSettled(_ context.Context, _ billing.OnPaymentSettledInput) error {
	return nil
}

func (e *LineEngine) CalculateLines(input billing.CalculateLinesInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice id is required")
	}

	if len(input.Lines) == 0 {
		return nil, fmt.Errorf("lines are required")
	}

	for _, stdLine := range input.Lines {
		generatedDetailedLines, err := e.service.ratingService.GenerateDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging generated detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}
	}

	return input.Lines, nil
}
