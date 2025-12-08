package meterexport

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Service manages exporting data from OpenMeter meters
type Service interface {
	// ExportSyntheticMeterData exports synthetic pre-aggregated events from OpenMeter. When ingested into a meter matching the descriptor, the resulted events will accurately reconstruct the meter histogram with WindowSize precision.
	// ExportSyntheticMeterData produces one event per WindowSize using the same event format as the storage layer.
	// This pre-aggregation is useful because while OpenMeter is designed to handle large event volumes, downstream systems usually don't care about the full granularity of all stored events.
	// This is a streaming operation, the result channel will be closed when the operation is complete. An error is only returned if the operation fails to start.
	// It is up to the caller to determine if a message on the error channel is critical and should stop the operation, which can be done by canceling the context.
	//
	// NOTE: Currently only SUM and COUNT meters are supported.
	// NOTE: GroupBy values are not yet supported.
	ExportSyntheticMeterData(ctx context.Context, config DataExportConfig, result chan<- streaming.RawEvent, err chan<- error) (TargetMeterDescriptor, error)
}

// TargetMeterDescriptor is a minimal MeterCreateInput which can accurately represent the exported data.
type TargetMeterDescriptor struct {
	Aggregation   meter.MeterAggregation
	EventType     string
	ValueProperty *string
}

type DataExportConfig struct {
	// Defines in what pre-aggregated windows the synthetic data will be exported in
	ExportWindowSize meter.WindowSize

	// The source meter to export data from
	MeterID models.NamespacedID

	// The period to export data for
	Period timeutil.StartBoundedPeriod
}

func (c DataExportConfig) Validate() error {
	var errs []error

	if c.ExportWindowSize == "" {
		errs = append(errs, errors.New("export window size is required"))
	}

	if c.MeterID.Namespace == "" {
		errs = append(errs, errors.New("meter namespace is required"))
	}

	if c.MeterID.ID == "" {
		errs = append(errs, errors.New("meter id is required"))
	}

	if err := c.Period.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
