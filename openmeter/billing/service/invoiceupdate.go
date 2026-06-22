package billingservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type mutableInvoiceLineDiff struct {
	billing.OnMutableInvoiceUpdateInput

	Unchanged []billing.GenericInvoiceLine
}

func (d mutableInvoiceLineDiff) IsEmpty() bool {
	return len(d.Created) == 0 && len(d.Updated) == 0 && len(d.Deleted) == 0
}

func diffMutableInvoiceLines(before, after billing.GenericInvoiceReader, createLineRouter billing.CreateLineRouter) (mutableInvoiceLineDiff, error) {
	if err := validateInvoiceReaderForDiff(before); err != nil {
		return mutableInvoiceLineDiff{}, fmt.Errorf("validating before invoice: %w", err)
	}

	if err := validateInvoiceReaderForDiff(after); err != nil {
		return mutableInvoiceLineDiff{}, fmt.Errorf("validating after invoice: %w", err)
	}

	beforeByID := lo.SliceToMap(before.GetGenericLines().OrEmpty(), func(line billing.GenericInvoiceLine) (string, billing.GenericInvoiceLine) {
		return line.GetID(), line
	})

	diff := mutableInvoiceLineDiff{}

	err := entitydiff.DiffByID(entitydiff.DiffByIDInput[billing.GenericInvoiceLine]{
		DBState:       before.GetGenericLines().OrEmpty(),
		ExpectedState: after.GetGenericLines().OrEmpty(),
		HandleCreate: func(item billing.GenericInvoiceLine) error {
			if item.GetDeletedAt() != nil {
				return nil
			}

			// Allocate a line engine for the new line if it doesn't have one yet.
			if item.GetEngine() == "" {
				engine, err := createLineRouter.GetLineEngineForCreateLine(item)
				if err != nil {
					return fmt.Errorf("getting line engine for new line: %w", err)
				}

				if err := engine.Validate(); err != nil {
					return fmt.Errorf("validating line engine type: %w", err)
				}

				item.SetEngine(engine)
			}

			diff.Created = append(diff.Created, item)
			return nil
		},
		HandleDelete: func(item billing.GenericInvoiceLine) error {
			beforeLine, ok := beforeByID[item.GetID()]
			if ok && beforeLine.GetDeletedAt() != nil {
				return nil
			}

			if item.GetDeletedAt() == nil {
				item.SetDeletedAt(lo.ToPtr(clock.Now()))
			}

			diff.Deleted = append(diff.Deleted, item)
			return nil
		},
		HandleUpdate: func(item entitydiff.DiffUpdate[billing.GenericInvoiceLine]) error {
			beforeLine := item.PersistedState
			afterLine := item.ExpectedState

			if beforeLine.GetDeletedAt() != nil {
				// Let's not allow restoring a deleted line.
				if afterLine.GetDeletedAt() == nil {
					return fmt.Errorf("line[%s]: cannot restore a deleted line", afterLine.GetID())
				}

				// Already-deleted lines can still be carried forward by system sync,
				// for example when a cancellation shrinks the deleted split-line period.
				// Keep them out of line-engine callbacks so delete side effects do not run twice.
				diff.Unchanged = append(diff.Unchanged, afterLine)
				return nil
			}

			engine := beforeLine.GetEngine()
			if engine == "" {
				return fmt.Errorf("line[%s]: line engine is required for updated line", beforeLine.GetID())
			}

			if err := engine.Validate(); err != nil {
				return fmt.Errorf("line[%s]: validating line engine type: %w", beforeLine.GetID(), err)
			}

			if afterLine.GetEngine() != "" && afterLine.GetEngine() != engine {
				return fmt.Errorf("line[%s]: line engine cannot be changed", afterLine.GetID())
			}

			if afterLine.GetDeletedAt() != nil {
				deletedLine, err := beforeLine.Clone()
				if err != nil {
					return fmt.Errorf("cloning deleted line[%s]: %w", beforeLine.GetID(), err)
				}

				deletedLine.SetDeletedAt(afterLine.GetDeletedAt())
				diff.Deleted = append(diff.Deleted, deletedLine)
				return nil
			}

			override, err := diffInvoiceLine(beforeLine, afterLine)
			if err != nil {
				return fmt.Errorf("line[%s]: building override: %w", afterLine.GetID(), err)
			}

			if override == nil {
				diff.Unchanged = append(diff.Unchanged, item.PersistedState)
				return nil
			}

			diff.Updated = append(diff.Updated, billing.InvoiceLineOverride{
				ExistingLine:   item.PersistedState,
				ChangesToApply: *override,
			})

			return nil
		},
	})
	if err != nil {
		return mutableInvoiceLineDiff{}, err
	}

	return diff, nil
}

func validateInvoiceReaderForDiff(invoice billing.GenericInvoiceReader) error {
	if invoice == nil {
		return fmt.Errorf("invoice is required")
	}

	if invoice.GetGenericLines().IsAbsent() {
		return fmt.Errorf("lines are required")
	}

	return nil
}

func diffInvoiceLine(before, after billing.GenericInvoiceLineReader) (*billing.ExistingLineOverride, error) {
	if before == nil {
		return nil, fmt.Errorf("before line is required")
	}

	if after == nil {
		return nil, fmt.Errorf("after line is required")
	}

	override := &billing.ExistingLineOverride{
		Name:        comparableOverride(before.GetName(), after.GetName()),
		Description: comparablePtrOverride(before.GetDescription(), after.GetDescription()),
		Metadata:    metadataOverride(before.GetMetadata(), after.GetMetadata()),
		Period:      equalerOverride(before.GetServicePeriod(), after.GetServicePeriod()),
		InvoiceAt:   equalerOverride(before.GetInvoiceAt(), after.GetInvoiceAt()),
		TaxConfig:   taxConfigOverride(before.GetTaxConfig(), after.GetTaxConfig()),
		Price:       equalerOverride(before.GetPrice(), after.GetPrice()),
		FeatureKey:  comparableOverride(before.GetFeatureKey(), after.GetFeatureKey()),
		Discounts:   equalerOverride(before.GetRateCardDiscounts(), after.GetRateCardDiscounts()),
	}

	if !override.IsPresent() {
		return nil, nil
	}

	return override, nil
}

func comparableOverride[T comparable](a, b T) mo.Option[T] {
	if a != b {
		return mo.Some(b)
	}

	return mo.None[T]()
}

func comparablePtrOverride[T comparable](a, b *T) mo.Option[*T] {
	if !equal.ComparablePtrEqual(a, b) {
		return mo.Some(b)
	}

	return mo.None[*T]()
}

func equalerOverride[T equal.Equaler[T]](a, b T) mo.Option[T] {
	if !a.Equal(b) {
		return mo.Some(b)
	}

	return mo.None[T]()
}

func taxConfigOverride(a, b *billing.TaxConfig) mo.Option[*billing.TaxConfig] {
	if !a.Equal(b) {
		return mo.Some(b)
	}

	return mo.None[*billing.TaxConfig]()
}

func metadataOverride(a, b models.Metadata) mo.Option[models.Metadata] {
	if a == nil && b == nil {
		return mo.None[models.Metadata]()
	}

	if a == nil || b == nil {
		return mo.Some(b)
	}

	if !a.Equal(b) {
		return mo.Some(b)
	}

	return mo.None[models.Metadata]()
}

type applyAPIInvoiceLineEditsInput struct {
	EditedInvoice billing.GenericInvoiceReader
	LineDiff      mutableInvoiceLineDiff
}

func (i applyAPIInvoiceLineEditsInput) Validate() error {
	var errs []error

	if i.EditedInvoice == nil {
		errs = append(errs, errors.New("edited invoice is required"))
	}

	return errors.Join(errs...)
}

func (s *Service) applyAPIInvoiceLineEdits(
	ctx context.Context,
	input applyAPIInvoiceLineEditsInput,
) (billing.GenericInvoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	edited, err := input.EditedInvoice.CloneAsGenericInvoice()
	if err != nil {
		return nil, fmt.Errorf("cloning edited invoice: %w", err)
	}

	lineDiff := input.LineDiff
	if lineDiff.IsEmpty() {
		return edited, nil
	}

	// The edited invoice should not be treated as the source of truth for lines
	// while engines are canonicalizing the diff-owned line changes.
	edited.UnsetLines()

	// Standard invoice line creation needs billing-owned IDs before line engines
	// run, because charge-backed creates attach realizations to those IDs.
	if len(lineDiff.Created) > 0 && edited.GetType() == billing.InvoiceTypeStandard {
		stdInvoice, err := edited.AsInvoice().AsStandardInvoice()
		if err != nil {
			return nil, fmt.Errorf("converting edited invoice to standard invoice: %w", err)
		}

		createdLines, err := s.preallocateCreatedStandardLines(ctx, preallocatedCreatedStandardLinesInput{
			Invoice:      stdInvoice.GetInvoiceID(),
			Currency:     stdInvoice.Currency,
			SchemaLevel:  stdInvoice.SchemaLevel,
			CreatedLines: lineDiff.Created,
		})
		if err != nil {
			return nil, fmt.Errorf("preallocating created standard invoice lines: %w", err)
		}

		lineDiff.Created = createdLines
	}

	lineDiff.Invoice = edited

	// Only after preallocation do we route created lines to engines, because
	// routing is allowed to stamp the created line's engine.
	changesByEngine, err := lineDiff.GroupByLineEngine()
	if err != nil {
		return nil, fmt.Errorf("grouping mutable invoice line changes by engine: %w", err)
	}

	resultingLines := make([]billing.GenericInvoiceLine, 0, len(lineDiff.Created)+len(lineDiff.Updated)+len(lineDiff.Deleted)+len(lineDiff.Unchanged))
	resultingLines = append(resultingLines, lineDiff.Unchanged...)
	// Deleted lines are canonical in the diff itself. Engines may reject API
	// deletes through the input, but they do not return deleted lines.
	resultingLines = append(resultingLines, lineDiff.Deleted...)

	for engineType, input := range changesByEngine {
		engine, err := s.lineEngines.Get(engineType)
		if err != nil {
			return nil, fmt.Errorf("getting engine %s: %w", engineType, err)
		}

		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf("validating API invoice line edit input for engine %s: %w", engine.GetLineEngineType(), err)
		}

		engineResult, err := engine.OnMutableInvoiceLinesEditedViaAPI(ctx, input)
		if err != nil {
			return nil, billing.NewLineEngineValidationError(engine, err)
		}

		// validate and merge created lines
		if err := validateLineEngineResult(input.Created, engineResult.CreatedLines); err != nil {
			return nil, fmt.Errorf("validating API invoice line edit created output for engine %s: %w", engine.GetLineEngineType(), err)
		}
		resultingLines = append(resultingLines, engineResult.CreatedLines...)

		// validate and merge updated lines
		if err := validateLineEngineResult(lo.Map(input.Updated, func(override billing.InvoiceLineOverride, _ int) billing.GenericInvoiceLine {
			return override.ExistingLine
		}), engineResult.UpdatedLines); err != nil {
			return nil, fmt.Errorf("validating API invoice line edit updated output for engine %s: %w", engine.GetLineEngineType(), err)
		}
		resultingLines = append(resultingLines, engineResult.UpdatedLines...)
	}

	if err := edited.SetLines(resultingLines); err != nil {
		return nil, fmt.Errorf("setting edited invoice lines: %w", err)
	}

	return edited, nil
}

type preallocatedCreatedStandardLinesInput struct {
	Invoice      billing.InvoiceID
	Currency     currencyx.Code
	SchemaLevel  int
	CreatedLines []billing.GenericInvoiceLine
}

func (i preallocatedCreatedStandardLinesInput) Validate() error {
	var errs []error

	if err := i.Invoice.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoice: %w", err))
	}

	if i.Currency != "" {
		err := i.Currency.Validate()
		if err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}

	if i.SchemaLevel <= 0 {
		errs = append(errs, errors.New("schema level is required"))
	}

	return errors.Join(errs...)
}

// preallocateCreatedStandardLines assigns stable IDs to newly-created standard
// standard invoice lines and persists provisional rows before line engines handle them,
// thus line engines can already put FK on these lines.
func (s *Service) preallocateCreatedStandardLines(
	ctx context.Context,
	input preallocatedCreatedStandardLinesInput,
) ([]billing.GenericInvoiceLine, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	standardLinesToCreate, err := lo.MapErr(input.CreatedLines, func(item billing.GenericInvoiceLine, idx int) (*billing.StandardLine, error) {
		standardLine, err := item.AsInvoiceLine().AsStandardLine()
		if err != nil {
			return nil, fmt.Errorf("converting created line[%s] to standard line: %w", item.GetID(), err)
		}

		if standardLine.ID == "" {
			standardLine.ID = ulid.Make().String()
		}

		standardLine.Namespace = input.Invoice.Namespace
		standardLine.InvoiceID = input.Invoice.ID
		if standardLine.Currency == "" {
			standardLine.Currency = input.Currency
		}

		return &standardLine, nil
	})
	if err != nil {
		return nil, fmt.Errorf("mapping created lines to standard lines: %w", err)
	}

	preallocatedLines, err := s.adapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace:   input.Invoice.Namespace,
		Lines:       standardLinesToCreate,
		SchemaLevel: input.SchemaLevel,
		InvoiceID:   input.Invoice.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("upserting provisional created lines: %w", err)
	}

	return lo.Map(preallocatedLines, func(line *billing.StandardLine, _ int) billing.GenericInvoiceLine {
		return line.AsGenericLine()
	}), nil
}

func validateLineEngineResult(expectedLines []billing.GenericInvoiceLine, actualLines []billing.GenericInvoiceLine) error {
	var errs []error

	if len(expectedLines) != len(actualLines) {
		expectedIDs := lo.FilterMap(expectedLines, func(line billing.GenericInvoiceLine, _ int) (string, bool) {
			id := line.GetID()

			return id, id != ""
		})

		actualIDs := lo.FilterMap(actualLines, func(line billing.GenericInvoiceLine, _ int) (string, bool) {
			id := line.GetID()

			return id, id != ""
		})

		errs = append(errs, fmt.Errorf("expected [nr_lines=%d,ids=%v] lines, got [nr_lines=%d,ids=%v]", len(expectedLines), expectedIDs, len(actualLines), actualIDs))
	}

	return errors.Join(errs...)
}

func (s *Service) dispatchSystemStandardLineDeletions(ctx context.Context, invoice billing.StandardInvoice, lineDiff mutableInvoiceLineDiff) error {
	if len(lineDiff.Deleted) == 0 {
		return nil
	}

	deletedLines, err := slicesx.MapWithErr(lineDiff.Deleted, func(line billing.GenericInvoiceLine) (*billing.StandardLine, error) {
		standardLine, err := line.AsInvoiceLine().AsStandardLine()
		if err != nil {
			return nil, fmt.Errorf("converting deleted line[%s] to standard line: %w", line.GetID(), err)
		}

		return &standardLine, nil
	})
	if err != nil {
		return err
	}

	input := billing.OnMutableStandardLinesDeletedInput{
		Invoice: invoice,
		Lines:   deletedLines,
	}
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating mutable standard lines deleted by system input: %w", err)
	}

	groupedLines, err := s.lineEngines.groupStandardLinesByEngine(input.Lines)
	if err != nil {
		return fmt.Errorf("grouping standard lines by engine: %w", err)
	}

	for _, grouped := range groupedLines {
		groupedInput := billing.OnMutableStandardLinesDeletedInput{
			Invoice: input.Invoice,
			Lines:   grouped.Lines,
		}

		if err := groupedInput.Validate(); err != nil {
			return fmt.Errorf("validating mutable standard lines deleted by system input for engine %s: %w", grouped.Engine.GetLineEngineType(), err)
		}

		if err := grouped.Engine.OnMutableStandardLinesDeletedBySystem(ctx, groupedInput); err != nil {
			return billing.NewLineEngineValidationError(grouped.Engine, err)
		}
	}

	return nil
}
