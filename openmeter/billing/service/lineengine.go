package billingservice

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type engineRegistry struct {
	mu               sync.RWMutex
	engines          map[billing.LineEngineType]billing.LineEngine
	createLineRouter billing.CreateLineRouter
}

func newEngineRegistry() *engineRegistry {
	return &engineRegistry{
		engines: make(map[billing.LineEngineType]billing.LineEngine),
	}
}

func (r *engineRegistry) Register(eng billing.LineEngine) error {
	if eng == nil {
		return fmt.Errorf("line engine is required")
	}

	engineType := eng.GetLineEngineType()
	if err := engineType.Validate(); err != nil {
		return fmt.Errorf("validating engine type: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.engines[engineType]; ok {
		return fmt.Errorf("engine %s is already registered", engineType)
	}

	r.engines[engineType] = eng
	return nil
}

func (r *engineRegistry) Get(engineType billing.LineEngineType) (billing.LineEngine, error) {
	if err := engineType.Validate(); err != nil {
		return nil, fmt.Errorf("validating engine type: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	eng, ok := r.engines[engineType]
	if !ok {
		return nil, fmt.Errorf("engine %s is not registered", engineType)
	}

	return eng, nil
}

func (r *engineRegistry) Deregister(engineType billing.LineEngineType) error {
	if err := engineType.Validate(); err != nil {
		return fmt.Errorf("validating engine type: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.engines[engineType]; !ok {
		return fmt.Errorf("engine %s is not registered", engineType)
	}

	delete(r.engines, engineType)

	return nil
}

func (r *engineRegistry) List() []billing.LineEngineType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return lo.Keys(r.engines)
}

func (r *engineRegistry) RegisterCreateLineRouter(router billing.CreateLineRouter) error {
	if router == nil {
		return fmt.Errorf("create line router is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.createLineRouter != nil {
		return fmt.Errorf("create line router is already registered")
	}

	r.createLineRouter = router

	return nil
}

func (r *engineRegistry) GetCreateLineRouter() billing.CreateLineRouter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.createLineRouter != nil {
		return r.createLineRouter
	}

	return billing.DefaultCreateLineRouter{}
}

func (r *engineRegistry) validateLineEngine(engineType billing.LineEngineType) error {
	return engineType.Validate()
}

func (r *engineRegistry) populateGatheringLineEngine(line *billing.GatheringLine) error {
	if line.Engine != "" {
		return r.validateLineEngine(line.Engine)
	}

	line.Engine = billing.LineEngineTypeInvoice
	return nil
}

func (r *engineRegistry) populateStandardLineEngine(line *billing.StandardLine) error {
	if line.Engine != "" {
		return r.validateLineEngine(line.Engine)
	}

	line.Engine = billing.LineEngineTypeInvoice
	return nil
}

type GatheringLinesWithEngine struct {
	Engine billing.LineEngine
	Lines  billing.GatheringLines
}

type StandardLinesWithEngine struct {
	Engine billing.LineEngine
	Lines  billing.StandardLines
}

type gatheringLineBillabilityResult struct {
	billing.IsLineBillableAsOfResult
	Engine billing.LineEngine
}

func (s *Service) areGatheringLinesBillableAsOf(ctx context.Context, input billing.AreLinesBillableAsOfInput) ([]gatheringLineBillabilityResult, error) {
	if len(input.Lines) == 0 {
		return []gatheringLineBillabilityResult{}, nil
	}

	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating billability input: %w", err)
	}

	linesWithEngines, err := s.lineEngines.groupGatheringLinesByEngine(input.Lines)
	if err != nil {
		return nil, fmt.Errorf("grouping gathering lines by engine: %w", err)
	}

	resultsByLineID := make(map[string]gatheringLineBillabilityResult, len(input.Lines))
	var errs []error
	for _, grouped := range linesWithEngines {
		engineType := grouped.Engine.GetLineEngineType()
		results, err := grouped.Engine.AreLinesBillableAsOf(ctx, billing.AreLinesBillableAsOfInput{
			Invoice:            input.Invoice,
			AsOf:               input.AsOf,
			ProgressiveBilling: input.ProgressiveBilling,
			Lines:              grouped.Lines,
		})
		if err != nil && !billing.IsValidationIssueOnly(err) {
			return nil, fmt.Errorf("checking line billability with engine %s: %w", engineType, err)
		}
		errs = append(errs, err)

		if len(results) != len(grouped.Lines) {
			return nil, fmt.Errorf("engine %s returned %d billability results for %d inputs", engineType, len(results), len(grouped.Lines))
		}

		for index, result := range results {
			lineID := grouped.Lines[index].ID
			if err := result.Validate(); err != nil {
				return nil, fmt.Errorf("validating line[%s] billability result from engine %s: %w", lineID, engineType, err)
			}

			resultsByLineID[lineID] = gatheringLineBillabilityResult{
				IsLineBillableAsOfResult: result,
				Engine:                   grouped.Engine,
			}
		}
	}

	results, err := lo.MapErr(input.Lines, func(line billing.GatheringLine, _ int) (gatheringLineBillabilityResult, error) {
		result, ok := resultsByLineID[line.ID]
		if !ok {
			return gatheringLineBillabilityResult{}, fmt.Errorf("billability result for line[%s] is missing", line.ID)
		}

		return result, nil
	})
	if err != nil {
		return nil, err
	}

	return results, errors.Join(errs...)
}

func (r *engineRegistry) groupGatheringLinesByEngine(lines billing.GatheringLines) ([]GatheringLinesWithEngine, error) {
	grouped := lo.GroupBy(lines, func(line billing.GatheringLine) billing.LineEngineType {
		return line.Engine
	})

	result := make([]GatheringLinesWithEngine, 0, len(grouped))
	for engineType, groupedLines := range grouped {
		eng, err := r.Get(engineType)
		if err != nil {
			return nil, fmt.Errorf("getting engine %s: %w", engineType, err)
		}

		result = append(result, GatheringLinesWithEngine{
			Engine: eng,
			Lines:  groupedLines,
		})
	}

	return result, nil
}

func (r *engineRegistry) groupStandardLinesByEngine(lines billing.StandardLines) ([]StandardLinesWithEngine, error) {
	grouped := lo.GroupBy(lines, func(line *billing.StandardLine) billing.LineEngineType {
		return line.Engine
	})

	result := make([]StandardLinesWithEngine, 0, len(grouped))
	for engineType, groupedLines := range grouped {
		eng, err := r.Get(engineType)
		if err != nil {
			return nil, fmt.Errorf("getting engine %s: %w", engineType, err)
		}

		result = append(result, StandardLinesWithEngine{
			Engine: eng,
			Lines:  groupedLines,
		})
	}

	return result, nil
}

func (s *Service) RegisterLineEngine(eng billing.LineEngine) error {
	return s.lineEngines.Register(eng)
}

func (s *Service) RegisterCreateLineRouter(router billing.CreateLineRouter) error {
	return s.lineEngines.RegisterCreateLineRouter(router)
}

func (s *Service) DeregisterLineEngine(engineType billing.LineEngineType) error {
	return s.lineEngines.Deregister(engineType)
}

func (s *Service) GetRegisteredLineEngines() []billing.LineEngineType {
	return s.lineEngines.List()
}

func (s *Service) OnUnsupportedCreditNote(ctx context.Context, input billing.OnUnsupportedCreditNoteInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating unsupported credit note input: %w", err)
	}

	groupedLines, err := s.lineEngines.groupStandardLinesByEngine(input.Lines)
	if err != nil {
		return fmt.Errorf("grouping standard lines by engine: %w", err)
	}

	for _, grouped := range groupedLines {
		groupedInput := billing.OnUnsupportedCreditNoteInput{
			Invoice: input.Invoice,
			Lines:   grouped.Lines,
		}

		if err := groupedInput.Validate(); err != nil {
			return fmt.Errorf("validating unsupported credit note input for engine %s: %w", grouped.Engine.GetLineEngineType(), err)
		}

		if err := grouped.Engine.OnUnsupportedCreditNote(ctx, groupedInput); err != nil {
			return billing.NewLineEngineValidationError(grouped.Engine, err)
		}
	}

	return nil
}
