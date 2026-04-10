package billingservice

import (
	"fmt"
	"sync"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type engineRegistry struct {
	mu      sync.RWMutex
	engines map[billing.LineEngineType]billing.LineEngine
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

func (s *Service) DeregisterLineEngine(engineType billing.LineEngineType) error {
	return s.lineEngines.Deregister(engineType)
}

func (s *Service) GetRegisteredLineEngines() []billing.LineEngineType {
	return s.lineEngines.List()
}
