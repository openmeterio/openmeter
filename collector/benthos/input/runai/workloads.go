package runai

import (
	"context"
	"fmt"
	"net/http"
	"time"
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

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetQueryParams(map[string]string{
			"limit":  fmt.Sprintf("%d", params.Limit),
			"offset": fmt.Sprintf("%d", params.Offset),
			// Matches phases: Creating, Initializing, Resuming, Pending, Deleting, Running, Updating, Stopping, Terminating
			"filterBy": "phase=$ing",
		}).
		SetResult(&ListWorkloadsResponse{}).
		Get("/api/v1/workloads")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to list workloads, status code: %d", resp.StatusCode())
	}

	s.logger.Tracef("list workloads response: %+v", resp.Result())
	result := resp.Result().(*ListWorkloadsResponse)
	return result, nil
}

// ListAllWorkloads lists all workloads.
func (s *Service) ListAllWorkloads(ctx context.Context) ([]Workload, error) {
	workloads := make([]Workload, 0)

	for {
		resp, err := s.ListWorkloads(ctx, ListWorkloadParams{
			Limit:  500,
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

	return workloads, nil
}
