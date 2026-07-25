package billingservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/syncx"
)

type mutableInvoiceLineDiff struct {
	billing.OnMutableInvoiceUpdateInput

	Unchanged []billing.GenericInvoiceLine
}

func (d mutableInvoiceLineDiff) IsEmpty() bool {
	return len(d.Created) == 0 && len(d.Updated) == 0 && len(d.Deleted) == 0
}

func (d mutableInvoiceLineDiff) Validate() error {
	if d.IsEmpty() {
		return nil
	}

	return d.OnMutableInvoiceUpdateInput.Validate()
}

func diffMutableInvoiceLines(
	before billing.GenericInvoiceReader,
	after billing.GenericInvoiceReader,
	source billing.ChangeSource,
	createLineRouter billing.CreateLineRouter,
) (mutableInvoiceLineDiff, error) {
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

			override, err := diffInvoiceLine(beforeLine, afterLine, source)
			if err != nil {
				return fmt.Errorf("line[%s]: building override: %w", afterLine.GetID(), err)
			}

			if override == nil {
				diff.Unchanged = append(diff.Unchanged, item.ExpectedState)
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

func (s *Service) diffMutableInvoiceLines(ctx context.Context, before billing.GenericInvoiceReader, after billing.GenericInvoiceReader, source billing.ChangeSource) (mutableInvoiceLineDiff, error) {
	if err := source.Validate(); err != nil {
		return mutableInvoiceLineDiff{}, err
	}

	if err := validateInvoiceReaderForDiff(before); err != nil {
		return mutableInvoiceLineDiff{}, fmt.Errorf("validating before invoice: %w", err)
	}

	if err := validateInvoiceReaderForDiff(after); err != nil {
		return mutableInvoiceLineDiff{}, fmt.Errorf("validating after invoice: %w", err)
	}

	defaultTaxCodeResolvers := s.defaultTaxCodeResolversForInvoiceUpdate(after)
	if source == billing.ChangeSourceAPIRequest {
		var err error
		before, err = s.invoiceWithSanitizedTaxConfigForDiff(ctx, defaultTaxCodeResolvers, before)
		if err != nil {
			return mutableInvoiceLineDiff{}, fmt.Errorf("sanitizing persisted invoice line tax configs for diff: %w", err)
		}

		after, err = s.invoiceWithSanitizedTaxConfigForDiff(ctx, defaultTaxCodeResolvers, after)
		if err != nil {
			return mutableInvoiceLineDiff{}, fmt.Errorf("sanitizing expected invoice line tax configs for diff: %w", err)
		}
	}

	diff, err := diffMutableInvoiceLines(before, after, source, s.lineEngines.GetCreateLineRouter())
	if err != nil {
		return mutableInvoiceLineDiff{}, err
	}

	if source == billing.ChangeSourceAPIRequest {
		for _, line := range diff.Created {
			line.SetManagedBy(billing.ManuallyManagedLine)
		}
	}

	diff.DefaultTaxCodeResolvers = defaultTaxCodeResolvers
	if err := diff.Validate(); err != nil {
		return mutableInvoiceLineDiff{}, err
	}

	return diff, nil
}

// invoiceWithSanitizedTaxConfigForDiff returns an invoice clone whose line tax
// configs are normalized for API edit diffing. The diff is the source of truth
// for line-engine inputs, so both the persisted and expected invoice states are
// compared in the same resolved representation: nil or identity-less tax configs
// use the provider-default tax code, explicit provider codes resolve to
// TaxCodeID, and diff equality ignores resolved TaxCode object snapshots.
func (s *Service) invoiceWithSanitizedTaxConfigForDiff(
	ctx context.Context,
	defaultTaxCodeResolvers billing.DefaultTaxCodeResolvers,
	invoice billing.GenericInvoiceReader,
) (billing.GenericInvoice, error) {
	if err := defaultTaxCodeResolvers.Validate(); err != nil {
		return nil, fmt.Errorf("default tax code resolvers: %w", err)
	}

	if invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}

	sanitizedInvoice, err := invoice.CloneAsGenericInvoice()
	if err != nil {
		return nil, fmt.Errorf("cloning invoice: %w", err)
	}

	if sanitizedInvoice.GetGenericLines().IsAbsent() {
		return sanitizedInvoice, nil
	}

	namespace := sanitizedInvoice.GetInvoiceID().Namespace
	lines := sanitizedInvoice.GetGenericLines().OrEmpty()
	for i, line := range lines {
		sanitizedLine, err := s.sanitizeInvoiceLineTaxConfigForDiff(ctx, namespace, line)
		if err != nil {
			return nil, fmt.Errorf("line[%s]: %w", line.GetID(), err)
		}

		lines[i] = sanitizedLine
	}

	if err := sanitizedInvoice.SetLines(lines); err != nil {
		return nil, fmt.Errorf("setting sanitized invoice lines: %w", err)
	}

	return sanitizedInvoice, nil
}

func (s *Service) sanitizeInvoiceLineTaxConfigForDiff(
	ctx context.Context,
	namespace string,
	line billing.GenericInvoiceLine,
) (billing.GenericInvoiceLine, error) {
	taxConfig, err := s.sanitizeTaxConfigForDiff(ctx, namespace, line.GetTaxConfig())
	if err != nil {
		return nil, err
	}

	invoiceLine := line.AsInvoiceLine()
	switch invoiceLine.Type() {
	case billing.InvoiceLineTypeStandard:
		standardLine, err := invoiceLine.AsStandardLine()
		if err != nil {
			return nil, err
		}

		standardLine.TaxConfig = taxConfig
		return standardLine.AsGenericLine(), nil

	case billing.InvoiceLineTypeGathering:
		gatheringLine, err := invoiceLine.AsGatheringLine()
		if err != nil {
			return nil, err
		}

		gatheringLine.TaxConfig = taxConfig.ToProductCatalog()
		return gatheringLine.AsGenericLine(), nil

	default:
		return nil, fmt.Errorf("unsupported invoice line type: %s", invoiceLine.Type())
	}
}

func (s *Service) sanitizeTaxConfigForDiff(
	ctx context.Context,
	namespace string,
	taxConfig *billing.TaxConfig,
) (*billing.TaxConfig, error) {
	var sanitized billing.TaxConfig
	if taxConfig != nil {
		sanitized = taxConfig.Clone()
	}

	if sanitized.TaxCodeID != nil && *sanitized.TaxCodeID == "" {
		sanitized.TaxCodeID = nil
	}

	if sanitized.TaxCodeID != nil {
		return &sanitized, nil
	}

	if sanitized.Stripe == nil || sanitized.Stripe.Code == "" {
		providerDefaultTaxCode, err := s.taxCodeService.GetTaxCodeByKey(ctx, taxcode.GetTaxCodeByKeyInput{
			Namespace: namespace,
			Key:       taxcode.ProviderDefaultTaxCodeKey,
		})
		if err != nil {
			return nil, fmt.Errorf("resolving provider default tax code: %w", err)
		}

		sanitized.TaxCodeID = lo.ToPtr(providerDefaultTaxCode.ID)

		if err := validateTaxConfigTaxCodeIDResolvedForDiff(&sanitized); err != nil {
			return nil, err
		}

		return &sanitized, nil
	}

	productCatalogTaxConfig := sanitized.ToProductCatalog()
	if err := productcatalog.ResolveTaxConfig(ctx, s.taxCodeService, namespace, productCatalogTaxConfig); err != nil {
		return nil, err
	}

	sanitized.TaxConfig = productCatalogTaxConfig.Clone()
	if err := validateTaxConfigTaxCodeIDResolvedForDiff(&sanitized); err != nil {
		return nil, err
	}

	return &sanitized, nil
}

func validateTaxConfigTaxCodeIDResolvedForDiff(taxConfig *billing.TaxConfig) error {
	if taxConfig != nil && taxConfig.TaxCodeID != nil && *taxConfig.TaxCodeID != "" {
		return nil
	}

	return billing.ValidationError{
		Err: models.NewGenericValidationError(errors.New("cannot resolve tax code id")),
	}
}

func validateInvoiceReaderForDiff(invoice billing.GenericInvoiceReader) error {
	if invoice == nil {
		return fmt.Errorf("invoice is required")
	}

	if invoice.GetGenericLines().IsAbsent() {
		return fmt.Errorf("lines are required")
	}

	for idx, line := range invoice.GetGenericLines().OrEmpty() {
		if line == nil {
			return fmt.Errorf("lines[%d]: line is nil", idx)
		}
	}

	return nil
}

func diffInvoiceLine(before, after billing.GenericInvoiceLineReader, source billing.ChangeSource) (*billing.ExistingLineOverride, error) {
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
		TaxConfig:   taxConfigOverride(before.GetTaxConfig(), after.GetTaxConfig(), source),
		Price:       equalerOverride(before.GetPrice(), after.GetPrice()),
		FeatureKey:  comparableOverride(before.GetFeatureKey(), after.GetFeatureKey()),
		Discounts:   equalerOverride(before.GetRateCardDiscounts(), after.GetRateCardDiscounts()),
	}

	// Standard lines have no invoice at (technically they have, but should not, so we should only diff for gathering lines)
	if invoiceAtReader, ok := after.(billing.InvoiceAtAccessor); ok {
		before, ok := before.(billing.InvoiceAtAccessor)
		if !ok {
			return nil, fmt.Errorf("before line is not an InvoiceAtAccessor")
		}

		override.InvoiceAt = equalerOverride(before.GetInvoiceAt(), invoiceAtReader.GetInvoiceAt())
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

func taxConfigOverride(a, b *billing.TaxConfig, source billing.ChangeSource) mo.Option[*billing.TaxConfig] {
	if source == billing.ChangeSourceAPIRequest {
		if taxConfigsEqualForInvoiceLineDiff(a, b) {
			return mo.None[*billing.TaxConfig]()
		}

		return mo.Some(b)
	}

	if taxConfigsEqual(a, b) {
		return mo.None[*billing.TaxConfig]()
	}

	return mo.Some(b)
}

func taxConfigsEqual(a, b *billing.TaxConfig) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return a.Equal(b)
}

// taxConfigsEqualForInvoiceLineDiff compares only the fields that represent an
// invoice-line tax edit after diff normalization. Both sides have already had
// provider-specific tax code intent resolved into TaxCodeID, so Behavior and
// TaxCodeID fully describe the editable state; resolved TaxCode snapshots are
// ignored because they are invoice snapshot metadata, not edit intent.
func taxConfigsEqualForInvoiceLineDiff(a, b *billing.TaxConfig) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if !equal.ComparablePtrEqual(a.Behavior, b.Behavior) {
		return false
	}

	return equal.ComparablePtrEqual(a.TaxCodeID, b.TaxCodeID)
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

	if len(lineDiff.Created) > 0 && edited.GetType() == billing.InvoiceTypeGathering {
		gatheringInvoice, err := edited.AsInvoice().AsGatheringInvoice()
		if err != nil {
			return nil, fmt.Errorf("converting edited invoice to gathering invoice: %w", err)
		}

		createdLines, err := s.preallocateCreatedGatheringLines(ctx, gatheringInvoice, lineDiff.Created)
		if err != nil {
			return nil, fmt.Errorf("preallocating created gathering invoice lines: %w", err)
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

	if err := s.validateAPIInvoiceLineEdits(ctx, changesByEngine); err != nil {
		return nil, err
	}

	resultingLines := make([]billing.GenericInvoiceLine, 0, len(lineDiff.Created)+len(lineDiff.Updated)+len(lineDiff.Deleted)+len(lineDiff.Unchanged))
	resultingLines = append(resultingLines, lineDiff.Unchanged...)

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

		if err := validateLineEngineResult(input.Created, engineResult.CreatedLines); err != nil {
			return nil, fmt.Errorf("validating API invoice line edit created output for engine %s: %w", engine.GetLineEngineType(), err)
		}
		// API-created inputs are stamped before engine dispatch, but engines may
		// return replacement line instances. Billing owns the API ownership
		// transition, so created outputs are stamped here as well.
		for _, line := range engineResult.CreatedLines {
			line.SetManagedBy(billing.ManuallyManagedLine)
		}
		resultingLines = append(resultingLines, engineResult.CreatedLines...)

		if err := validateLineEngineResult(lo.Map(input.Updated, func(override billing.InvoiceLineOverride, _ int) billing.GenericInvoiceLine {
			return override.ExistingLine
		}), engineResult.UpdatedLines); err != nil {
			return nil, fmt.Errorf("validating API invoice line edit updated output for engine %s: %w", engine.GetLineEngineType(), err)
		}
		// Updated lines are stamped after the engine runs. This lets engines see
		// whether the API edit is system/subscription -> manual or manual -> manual,
		// while billing still owns the API ownership transition.
		for _, line := range engineResult.UpdatedLines {
			line.SetManagedBy(billing.ManuallyManagedLine)
		}
		resultingLines = append(resultingLines, engineResult.UpdatedLines...)
	}

	// Deleted lines are canonical in the diff itself and engines do not return
	// replacements for them. Stamp API ownership after engines had a chance to
	// inspect the previous ownership edge.
	for _, line := range lineDiff.Deleted {
		line.SetManagedBy(billing.ManuallyManagedLine)
	}
	resultingLines = append(resultingLines, lineDiff.Deleted...)

	if err := edited.SetLines(resultingLines); err != nil {
		return nil, fmt.Errorf("setting edited invoice lines: %w", err)
	}

	return edited, nil
}

func (s *Service) validateAPIInvoiceLineEdits(ctx context.Context, changesByEngine map[billing.LineEngineType]billing.OnMutableInvoiceUpdateInput) error {
	var errs []error

	for engineType, input := range changesByEngine {
		engine, err := s.lineEngines.Get(engineType)
		if err != nil {
			errs = append(errs, fmt.Errorf("getting engine %s: %w", engineType, err))
			continue
		}

		if err := input.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("validating API invoice line edit input for engine %s: %w", engine.GetLineEngineType(), err))
			continue
		}

		if err := engine.ValidateMutableInvoiceLineEditViaAPI(ctx, input); err != nil {
			errs = append(errs, billing.NewLineEngineValidationError(engine, err))
		}
	}

	return errors.Join(errs...)
}

func (s *Service) defaultTaxCodeResolversForInvoiceUpdate(invoice billing.GenericInvoiceReader) billing.DefaultTaxCodeResolvers {
	namespace := invoice.GetInvoiceID().Namespace
	getCustomerOverride := syncx.OnceValues(func(ctx context.Context) (billing.CustomerOverrideWithDetails, error) {
		return s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: invoice.GetCustomerID(),
		})
	})
	getOrganizationDefaultTaxCodes := syncx.OnceValues(func(ctx context.Context) (taxcode.OrganizationDefaultTaxCodes, error) {
		return s.taxCodeService.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: namespace,
		})
	})

	return billing.DefaultTaxCodeResolvers{
		Invoicing: func(ctx context.Context) (string, error) {
			return s.defaultInvoicingTaxCodeIDForInvoiceUpdate(ctx, invoice, getCustomerOverride, getOrganizationDefaultTaxCodes)
		},
		CreditGrant: func(ctx context.Context) (string, error) {
			defaults, err := getOrganizationDefaultTaxCodes(ctx)
			if err != nil {
				return "", fmt.Errorf("getting organization default tax codes: %w", err)
			}

			return defaults.CreditGrantTaxCodeID, nil
		},
	}
}

func (s *Service) defaultInvoicingTaxCodeIDForInvoiceUpdate(
	ctx context.Context,
	invoice billing.GenericInvoiceReader,
	getCustomerOverride func(context.Context) (billing.CustomerOverrideWithDetails, error),
	getOrganizationDefaultTaxCodes func(context.Context) (taxcode.OrganizationDefaultTaxCodes, error),
) (string, error) {
	namespace := invoice.GetInvoiceID().Namespace

	if invoice.GetType() == billing.InvoiceTypeStandard {
		standardInvoice, err := invoice.AsInvoice().AsStandardInvoice()
		if err != nil {
			return "", fmt.Errorf("getting standard invoice: %w", err)
		}

		taxCodeID, err := s.taxCodeIDWithBackfill(ctx, namespace, standardInvoice.Workflow.Config.Invoicing.DefaultTaxConfig)
		if err != nil {
			return "", fmt.Errorf("resolving standard invoice default tax config: %w", err)
		}
		if taxCodeID != "" {
			return taxCodeID, nil
		}
	}

	customerOverride, err := getCustomerOverride(ctx)
	if err != nil {
		return "", fmt.Errorf("getting customer billing profile: %w", err)
	}

	taxCodeID, err := s.taxCodeIDWithBackfill(ctx, namespace, customerOverride.MergedProfile.WorkflowConfig.Invoicing.DefaultTaxConfig)
	if err != nil {
		return "", fmt.Errorf("resolving customer billing profile default tax config: %w", err)
	}
	if taxCodeID != "" {
		return taxCodeID, nil
	}

	defaults, err := getOrganizationDefaultTaxCodes(ctx)
	if err != nil {
		return "", fmt.Errorf("getting organization default tax codes: %w", err)
	}

	return defaults.InvoicingTaxCodeID, nil
}

func (s *Service) taxCodeIDWithBackfill(ctx context.Context, namespace string, taxConfig *productcatalog.TaxConfig) (string, error) {
	if taxConfig == nil {
		return "", nil
	}

	resolved := taxConfig.Clone()
	if err := s.resolveDefaultTaxCode(ctx, namespace, &resolved); err != nil {
		return "", err
	}

	return lo.FromPtr(resolved.TaxCodeID), nil
}

type preallocatedCreatedStandardLinesInput struct {
	Invoice      billing.InvoiceID
	Currency     currencyx.FiatCode
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

// preallocateCreatedGatheringLines assigns and persists billing-owned identity
// for API-created gathering lines before line engines create charge records.
// The provisional row has no charge ID yet; billing upserts the canonical
// charge-backed line after the engine attaches the created charge.
func (s *Service) preallocateCreatedGatheringLines(ctx context.Context, invoice billing.GatheringInvoice, createdLines []billing.GenericInvoiceLine) ([]billing.GenericInvoiceLine, error) {
	if s.adapter == nil {
		return nil, fmt.Errorf("billing adapter is required")
	}

	preallocatedLines, err := slicesx.MapWithErr(createdLines, func(item billing.GenericInvoiceLine) (billing.GatheringLine, error) {
		gatheringLine, err := item.AsInvoiceLine().AsGatheringLine()
		if err != nil {
			return billing.GatheringLine{}, fmt.Errorf("converting created line[%s] to gathering line: %w", item.GetID(), err)
		}

		if gatheringLine.ID == "" {
			gatheringLine.ID = ulid.Make().String()
		}

		if gatheringLine.UBPConfigID == "" {
			gatheringLine.UBPConfigID = ulid.Make().String()
		}

		gatheringLine.Namespace = invoice.Namespace
		gatheringLine.InvoiceID = invoice.ID
		if gatheringLine.Currency == "" {
			gatheringLine.Currency = invoice.Currency
		}

		return gatheringLine, nil
	})
	if err != nil {
		return nil, err
	}

	invoice.Lines = billing.NewGatheringInvoiceLines(preallocatedLines)
	if err := s.adapter.UpdateGatheringInvoice(ctx, invoice); err != nil {
		return nil, fmt.Errorf("upserting provisional created gathering lines: %w", err)
	}

	return lo.Map(preallocatedLines, func(line billing.GatheringLine, _ int) billing.GenericInvoiceLine {
		return line.AsGenericLine()
	}), nil
}

func validateLineEngineResult(expectedLines []billing.GenericInvoiceLine, actualLines []billing.GenericInvoiceLine) error {
	var errs []error

	expectedIDs := lo.FilterMap(expectedLines, func(line billing.GenericInvoiceLine, idx int) (string, bool) {
		if line == nil {
			errs = append(errs, fmt.Errorf("expected line[%d]: line is nil", idx))
			return "", false
		}

		id := line.GetID()
		return id, id != ""
	})

	actualIDs := lo.FilterMap(actualLines, func(line billing.GenericInvoiceLine, idx int) (string, bool) {
		if line == nil {
			errs = append(errs, fmt.Errorf("line[%d]: line is nil", idx))
			return "", false
		}

		id := line.GetID()
		return id, id != ""
	})

	if len(expectedLines) != len(actualLines) {
		errs = append(errs, fmt.Errorf("expected [nr_lines=%d,ids=%v] lines, got [nr_lines=%d,ids=%v]", len(expectedLines), expectedIDs, len(actualLines), actualIDs))
	}

	if len(expectedIDs) != len(lo.Uniq(expectedIDs)) {
		errs = append(errs, fmt.Errorf("expected line ids must be unique: %v", expectedIDs))
	}

	if len(actualIDs) != len(lo.Uniq(actualIDs)) {
		errs = append(errs, fmt.Errorf("actual line ids must be unique: %v", actualIDs))
	}

	missingIDs, unexpectedIDs := lo.Difference(lo.Uniq(expectedIDs), lo.Uniq(actualIDs))
	if len(missingIDs) > 0 {
		errs = append(errs, fmt.Errorf("missing line ids: %v", missingIDs))
	}

	if len(unexpectedIDs) > 0 {
		errs = append(errs, fmt.Errorf("unexpected line ids: %v", unexpectedIDs))
	}

	return errors.Join(errs...)
}

func (s *Service) dispatchAPIStandardLineDeletions(ctx context.Context, invoice billing.StandardInvoice, deletedLinesIn billing.StandardLines) error {
	changesByEngine, err := s.groupAPIStandardLineDeletionsByEngine(invoice, deletedLinesIn)
	if err != nil {
		return err
	}

	for engineType, groupedInput := range changesByEngine {
		engine, err := s.lineEngines.Get(engineType)
		if err != nil {
			return fmt.Errorf("getting engine %s: %w", engineType, err)
		}

		engineResult, err := engine.OnMutableInvoiceLinesEditedViaAPI(ctx, groupedInput)
		if err != nil {
			return billing.NewLineEngineValidationError(engine, err)
		}

		if err := validateLineEngineResult(groupedInput.Created, engineResult.CreatedLines); err != nil {
			return fmt.Errorf("validating API invoice line delete created output for engine %s: %w", engine.GetLineEngineType(), err)
		}

		if err := validateLineEngineResult(lo.Map(groupedInput.Updated, func(override billing.InvoiceLineOverride, _ int) billing.GenericInvoiceLine {
			return override.ExistingLine
		}), engineResult.UpdatedLines); err != nil {
			return fmt.Errorf("validating API invoice line delete updated output for engine %s: %w", engine.GetLineEngineType(), err)
		}
	}

	return nil
}

func (s *Service) validateAPIStandardLineDeletions(ctx context.Context, invoice billing.StandardInvoice, deletedLinesIn billing.StandardLines) error {
	changesByEngine, err := s.groupAPIStandardLineDeletionsByEngine(invoice, deletedLinesIn)
	if err != nil {
		return err
	}

	return s.validateAPIInvoiceLineEdits(ctx, changesByEngine)
}

func (s *Service) groupAPIStandardLineDeletionsByEngine(invoice billing.StandardInvoice, deletedLinesIn billing.StandardLines) (map[billing.LineEngineType]billing.OnMutableInvoiceUpdateInput, error) {
	if len(deletedLinesIn) == 0 {
		return nil, nil
	}

	var errs []error

	for _, stdLine := range deletedLinesIn {
		if stdLine == nil {
			errs = append(errs, errors.New("line is nil"))
			continue
		}

		if err := s.lineEngines.populateStandardLineEngine(stdLine); err != nil {
			errs = append(errs, fmt.Errorf("line[%s]: inferring engine: %w", stdLine.GetID(), err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	input := billing.OnMutableInvoiceUpdateInput{
		Invoice:                 invoice,
		DefaultTaxCodeResolvers: s.defaultTaxCodeResolversForInvoiceUpdate(invoice),
		Deleted:                 deletedLinesIn.AsGenericLines(),
	}
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating mutable invoice line delete input: %w", err)
	}

	changesByEngine, err := input.GroupByLineEngine()
	if err != nil {
		return nil, fmt.Errorf("grouping mutable invoice line deletes by engine: %w", err)
	}

	return changesByEngine, nil
}

func (s *Service) dispatchSystemStandardLineDeletions(ctx context.Context, invoice billing.StandardInvoice, deletedLinesIn []billing.GenericInvoiceLine) error {
	if len(deletedLinesIn) == 0 {
		return nil
	}

	deletedLines, err := slicesx.MapWithErr(deletedLinesIn, func(line billing.GenericInvoiceLine) (*billing.StandardLine, error) {
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
