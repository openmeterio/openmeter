package runai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

type MetricType string

// Workload metrics
const (
	WorkloadMetricTypeCPULimitCores    MetricType = "CPU_LIMIT_CORES"
	WorkloadMetricTypeCPUMemoryLimit   MetricType = "CPU_MEMORY_LIMIT_BYTES"
	WorkloadMetricTypeCPUMemoryRequest MetricType = "CPU_MEMORY_REQUEST_BYTES"
	WorkloadMetricTypeCPUMemoryUsage   MetricType = "CPU_MEMORY_USAGE_BYTES"
	WorkloadMetricTypeCPURequestCores  MetricType = "CPU_REQUEST_CORES"
	WorkloadMetricTypeCPUUsageCores    MetricType = "CPU_USAGE_CORES"
	WorkloadMetricTypeGPUAllocation    MetricType = "GPU_ALLOCATION"
	WorkloadMetricTypeGPUMemoryRequest MetricType = "GPU_MEMORY_REQUEST_BYTES"
	WorkloadMetricTypeGPUMemoryUsage   MetricType = "GPU_MEMORY_USAGE_BYTES"
	WorkloadMetricTypeGPUUtilization   MetricType = "GPU_UTILIZATION"
	WorkloadMetricTypePodCount         MetricType = "POD_COUNT"
	WorkloadMetricTypeRunningPodCount  MetricType = "RUNNING_POD_COUNT"
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
	Metrics Metrics `json:"metrics"`
}

// GetAllWorkloadWithMetrics gets metrics of all workloads.
func (s *Service) GetAllWorkloadWithMetrics(ctx context.Context, params MeasurementParams) ([]WorkloadWithMetrics, error) {
	workloads, err := s.ListAllWorkloads(ctx)
	if err != nil {
		return nil, err
	}

	workloadsWithMetrics := make([]WorkloadWithMetrics, len(workloads))
	for i, workload := range workloads {
		metrics := Metrics{
			Timestamp: params.StartTime,
			Values:    make(map[MetricType]float64),
		}

		// We chunk the metric types to not exceed the max number of metrics per request
		for _, metricTypes := range lo.Chunk(params.MetricType, 9) {
			m, err := s.GetWorkloadMetrics(ctx, workload.ID, MeasurementParams{
				MetricType: metricTypes,
				StartTime:  params.StartTime,
				EndTime:    params.EndTime,
			})
			if err != nil {
				return nil, err
			}

			metrics.Timestamp = m.Timestamp
			for mt, v := range m.Values {
				metrics.Values[mt] = v
			}
		}

		workloadsWithMetrics[i] = WorkloadWithMetrics{
			Workload: workload,
			Metrics:  metrics,
		}
	}

	return workloadsWithMetrics, nil
}

// Pod metrics
const (
	PodMetricTypeCPUMemoryUsageBytes                 MetricType = "CPU_MEMORY_USAGE_BYTES"
	PodMetricTypeCPUUsageCores                       MetricType = "CPU_USAGE_CORES"
	PodMetricTypeGPUFP16EngineActivityPerGPU         MetricType = "GPU_FP16_ENGINE_ACTIVITY_PER_GPU"
	PodMetricTypeGPUFP32EngineActivityPerGPU         MetricType = "GPU_FP32_ENGINE_ACTIVITY_PER_GPU"
	PodMetricTypeGPUFP64EngineActivityPerGPU         MetricType = "GPU_FP64_ENGINE_ACTIVITY_PER_GPU"
	PodMetricTypeGPUGraphicsEngineActivityPerGPU     MetricType = "GPU_GRAPHICS_ENGINE_ACTIVITY_PER_GPU"
	PodMetricTypeGPUMemoryBandwidthUtilizationPerGPU MetricType = "GPU_MEMORY_BANDWIDTH_UTILIZATION_PER_GPU"
	PodMetricTypeGPUMemoryUsageBytes                 MetricType = "GPU_MEMORY_USAGE_BYTES"
	PodMetricTypeGPUMemoryUsageBytesPerGPU           MetricType = "GPU_MEMORY_USAGE_BYTES_PER_GPU"
	PodMetricTypeGPUNVLinkReceivedBandwidthPerGPU    MetricType = "GPU_NVLINK_RECEIVED_BANDWIDTH_PER_GPU"
	PodMetricTypeGPUNVLinkTransmittedBandwidthPerGPU MetricType = "GPU_NVLINK_TRANSMITTED_BANDWIDTH_PER_GPU"
	PodMetricTypeGPUPCieReceivedBandwidthPerGPU      MetricType = "GPU_PCIE_RECEIVED_BANDWIDTH_PER_GPU"
	PodMetricTypeGPUPCieTransmittedBandwidthPerGPU   MetricType = "GPU_PCIE_TRANSMITTED_BANDWIDTH_PER_GPU"
	PodMetricTypeGPUSMActivityPerGPU                 MetricType = "GPU_SM_ACTIVITY_PER_GPU"
	PodMetricTypeGPUSMOccupancyPerGPU                MetricType = "GPU_SM_OCCUPANCY_PER_GPU"
	PodMetricTypeGPUSwapMemoryBytesPerGPU            MetricType = "GPU_SWAP_MEMORY_BYTES_PER_GPU"
	PodMetricTypeGPUTensorActivityPerGPU             MetricType = "GPU_TENSOR_ACTIVITY_PER_GPU"
	PodMetricTypeGPUUtilization                      MetricType = "GPU_UTILIZATION"
	PodMetricTypeGPUUtilizationPerGPU                MetricType = "GPU_UTILIZATION_PER_GPU"
)

// GetPodMetrics gets metrics of a pod.
func (s *Service) GetPodMetrics(ctx context.Context, workloadID string, podID string, params MeasurementParams) (Metrics, error) {
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
		Get(fmt.Sprintf("/api/v1/workloads/%s/pods/%s/metrics", workloadID, podID))
	if err != nil {
		return m, err
	}

	if resp.StatusCode() != http.StatusOK {
		return m, fmt.Errorf("failed to get pod metrics, status code: %d", resp.StatusCode())
	}

	result := resp.Result().(*MeasurementResponse)
	if result == nil {
		return m, fmt.Errorf("failed to get pod metrics, result is nil")
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

type PodWithMetrics struct {
	Pod
	Metrics Metrics `json:"metrics"`
}

// GetAllPodWithMetrics gets metrics of all pods.
func (s *Service) GetAllPodWithMetrics(ctx context.Context, params MeasurementParams) ([]PodWithMetrics, error) {
	pods, err := s.ListAllPods(ctx)
	if err != nil {
		return nil, err
	}

	podsWithMetrics := make([]PodWithMetrics, len(pods))
	for i, pod := range pods {
		metrics := Metrics{
			Timestamp: params.StartTime,
			Values:    make(map[MetricType]float64),
		}

		// We chunk the metric types to not exceed the max number of metrics per request
		for _, metricTypes := range lo.Chunk(params.MetricType, 9) {
			m, err := s.GetPodMetrics(ctx, pod.WorkloadID, pod.ID, MeasurementParams{
				MetricType: metricTypes,
				StartTime:  params.StartTime,
				EndTime:    params.EndTime,
			})
			if err != nil {
				return nil, err
			}

			metrics.Timestamp = m.Timestamp
			for mt, v := range m.Values {
				metrics.Values[mt] = v
			}
		}

		podsWithMetrics[i] = PodWithMetrics{
			Pod:     pod,
			Metrics: metrics,
		}
	}

	return podsWithMetrics, nil
}

type ResourceWithMetrics struct {
	resourceType string
	Pod          *Pod
	Workload     *Workload
	Metrics      Metrics
}

func ResourceWithMetricsFromWorkload(workload *WorkloadWithMetrics) ResourceWithMetrics {
	return ResourceWithMetrics{
		resourceType: "workload",
		Workload:     &workload.Workload,
		Metrics:      workload.Metrics,
	}
}

func ResourceWithMetricsFromPod(pod *PodWithMetrics) ResourceWithMetrics {
	return ResourceWithMetrics{
		resourceType: "pod",
		Pod:          &pod.Pod,
		Metrics:      pod.Metrics,
	}
}

func (r *ResourceWithMetrics) AsWorkloadWithMetrics() *WorkloadWithMetrics {
	if r.resourceType == "workload" && r.Workload != nil {
		return &WorkloadWithMetrics{
			Workload: *r.Workload,
			Metrics:  r.Metrics,
		}
	}

	return nil
}

func (r *ResourceWithMetrics) AsPodWithMetrics() *PodWithMetrics {
	if r.resourceType == "pod" && r.Pod != nil {
		return &PodWithMetrics{
			Pod:     *r.Pod,
			Metrics: r.Metrics,
		}
	}

	return nil
}

func (r ResourceWithMetrics) MarshalJSON() ([]byte, error) {
	switch r.resourceType {
	case "workload":
		return json.Marshal(r.AsWorkloadWithMetrics())
	case "pod":
		return json.Marshal(r.AsPodWithMetrics())
	}

	return nil, fmt.Errorf("unknown resource type: %s", r.resourceType)
}
