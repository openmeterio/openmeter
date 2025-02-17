package runai

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

type MetricType string

const (
	MetricTypeCPULimitCores    MetricType = "CPU_LIMIT_CORES"
	MetricTypeCPUMemoryLimit   MetricType = "CPU_MEMORY_LIMIT_BYTES"
	MetricTypeCPUMemoryRequest MetricType = "CPU_MEMORY_REQUEST_BYTES"
	MetricTypeCPUMemoryUsage   MetricType = "CPU_MEMORY_USAGE_BYTES"
	MetricTypeCPURequestCores  MetricType = "CPU_REQUEST_CORES"
	MetricTypeCPUUsageCores    MetricType = "CPU_USAGE_CORES"
	MetricTypeGPUAllocation    MetricType = "GPU_ALLOCATION"
	MetricTypeGPUMemoryRequest MetricType = "GPU_MEMORY_REQUEST_BYTES"
	MetricTypeGPUMemoryUsage   MetricType = "GPU_MEMORY_USAGE_BYTES"
	MetricTypeGPUUtilization   MetricType = "GPU_UTILIZATION"
	MetricTypePodCount         MetricType = "POD_COUNT"
	MetricTypeRunningPodCount  MetricType = "RUNNING_POD_COUNT"
)

type MeasurementParams struct {
	MetricType []MetricType `json:"metricType"`
	StartTime  time.Time    `json:"start"`
	EndTime    time.Time    `json:"end"`
}

type Measurement struct {
	Type   string `json:"type"`
	Values []struct {
		Value     string    `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	} `json:"values"`
}

type MeasurementResponse struct {
	Measurements []Measurement `json:"measurements"`
}

type Metrics struct {
	Timestamp time.Time              `json:"timestamp"`
	Values    map[MetricType]float64 `json:"values"`
}

// GetWorkloadMetrics gets metrics of a workload.
func (s *Service) GetWorkloadMetrics(ctx context.Context, workloadID string, params MeasurementParams) (Metrics, error) {
	m := Metrics{
		Timestamp: params.StartTime,
		Values:    make(map[MetricType]float64),
	}

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetQueryParams(map[string]string{
			"metricType": strings.Join(lo.Map(params.MetricType, func(metricType MetricType, _ int) string {
				return string(metricType)
			}), ","),
			"start":           params.StartTime.Format(time.RFC3339),
			"end":             params.EndTime.Format(time.RFC3339),
			"numberOfSamples": "1",
		}).
		SetResult(&MeasurementResponse{}).
		Get(fmt.Sprintf("/api/v1/workloads/%s/metrics", workloadID))
	if err != nil {
		return m, err
	}

	if resp.StatusCode() != http.StatusOK {
		return m, fmt.Errorf("failed to get workload metrics, status code: %d", resp.StatusCode())
	}

	result := resp.Result().(*MeasurementResponse)
	if result == nil {
		return m, fmt.Errorf("failed to get workload metrics, result is nil")
	}

	for _, measurement := range result.Measurements {
		if len(measurement.Values) > 0 {
			v, err := strconv.ParseFloat(measurement.Values[0].Value, 64)
			if err != nil {
				return m, fmt.Errorf("failed to parse metric value: %w", err)
			}

			m.Values[MetricType(measurement.Type)] = v
		} else {
			m.Values[MetricType(measurement.Type)] = 0
		}
	}

	return m, nil
}

type WorkloadWithMetrics struct {
	Workload
	Timestamp time.Time `json:"timestamp"`
	Metrics   Metrics   `json:"metrics"`
}

func (s *Service) GetAllWorkloadWithMetrics(ctx context.Context, params MeasurementParams) ([]WorkloadWithMetrics, error) {
	workloads, err := s.ListAllWorkloads(ctx)
	if err != nil {
		return nil, err
	}

	workloadsWithMetrics := make([]WorkloadWithMetrics, len(workloads))
	for i, workload := range workloads {
		metrics, err := s.GetWorkloadMetrics(ctx, workload.ID, params)
		if err != nil {
			return nil, err
		}

		workloadsWithMetrics[i] = WorkloadWithMetrics{
			Workload: workload,
			Metrics:  metrics,
		}
	}

	return workloadsWithMetrics, nil
}
