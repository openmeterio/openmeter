package runai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/samber/lo"
)

// TODO: add fields as needed, see: https://api-docs.run.ai/latest/tag/Workloads#operation/get_workloads
// Workload represents a Run:ai workload.
type Workload struct {
	TenantID                   int    `json:"tenantId"`
	Type                       string `json:"type"`
	Namespace                  string `json:"namespace"`
	Name                       string `json:"name"`
	ID                         string `json:"id"`
	PriorityClassName          string `json:"priorityClassName"`
	SubmittedBy                string `json:"submittedBy"`
	ClusterID                  string `json:"clusterId"`
	ProjectName                string `json:"projectName"`
	ProjectID                  string `json:"projectId"`
	DepartmentName             string `json:"departmentName"`
	DepartmentID               string `json:"departmentId"`
	RunningPods                int    `json:"runningPods"`
	WorkloadRequestedResources struct {
		GPURequestType string `json:"gpuRequestType"`
		GPU            struct {
			Limit   float64 `json:"limit"`
			Request float64 `json:"request"`
		} `json:"gpu"`
		GPUMemory struct {
			Limit   string `json:"limit"`
			Request string `json:"request"`
		} `json:"gpuMemory"`
		CPU struct {
			Limit   float64 `json:"limit"`
			Request float64 `json:"request"`
		} `json:"cpu"`
		CPUMemory struct {
			Limit   string `json:"limit"`
			Request string `json:"request"`
		} `json:"cpuMemory"`
	} `json:"workloadRequestedResources"`
	AllocatedResources struct {
		GPU       float64 `json:"gpu"`
		GPUMemory string  `json:"gpuMemory"`
		CPU       float64 `json:"cpu"`
		CPUMemory string  `json:"cpuMemory"`
	} `json:"allocatedResources"`
	Phase         string `json:"phase"`
	Preemptible   *bool  `json:"preemptible,omitempty"`
	RequestedPods struct {
		Count int `json:"count"`
	} `json:"requestedPods"`
	IdleGPUs          *int      `json:"idleGpus,omitempty"`
	IdleAllocatedGPUs *float64  `json:"idleAllocatedGpus,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
}

type ListWorkloadParams struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type ListWorkloadsResponse struct {
	Workloads []Workload `json:"workloads"`
	Next      int        `json:"next"`
}

// ListWorkloads lists workloads.
func (s *Service) ListWorkloads(ctx context.Context, params ListWorkloadParams) (*ListWorkloadsResponse, error) {
	if params.Limit > 500 {
		return nil, fmt.Errorf("limit must be less than 500")
	}

	// Filter out workloads where phase is in the list
	filteredOutPhases := []string{
		"Creating",
		"Initializing",
		"Resuming",
		"Pending",
		// "Deleting",
		// "Running",
		"Updating",
		"Stopped",
		"Stopping",
		// "Degraded",
		"Failed",
		"Completed",
		"Terminating",
		"Unknown",
	}

	filterBy := strings.Join(lo.Map(filteredOutPhases, func(p string, _ int) string {
		return fmt.Sprintf("phase!=%s", p)
	}), ",")

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetQueryParams(map[string]string{
			"limit":    fmt.Sprintf("%d", params.Limit),
			"offset":   fmt.Sprintf("%d", params.Offset),
			"filterBy": filterBy,
		}).
		SetResult(&ListWorkloadsResponse{}).
		Get("/api/v1/workloads")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to list workloads, status code: %d", resp.StatusCode())
	}

	result, ok := resp.Result().(*ListWorkloadsResponse)
	if !ok {
		return nil, fmt.Errorf("failed to parse list workloads response")
	}

	// Filter out workloads where running pod count is 0
	result.Workloads = lo.Filter(result.Workloads, func(w Workload, _ int) bool {
		return w.RunningPods > 0
	})

	j, err := json.Marshal(result)
	if err == nil {
		s.logger.Tracef("list workloads response: %s", string(j))
	}

	return result, nil
}

// ListAllWorkloads lists all workloads.
func (s *Service) ListAllWorkloads(ctx context.Context) ([]Workload, error) {
	workloads := make([]Workload, 0)

	for {
		resp, err := s.ListWorkloads(ctx, ListWorkloadParams{
			Limit:  s.pageSize,
			Offset: len(workloads),
		})
		if err != nil {
			return nil, err
		}

		workloads = append(workloads, resp.Workloads...)

		if resp.Next == 0 {
			break
		}
	}

	s.resourceTypeMetrics.Set(int64(len(workloads)), "workload")

	return workloads, nil
}
