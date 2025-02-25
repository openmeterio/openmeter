package runai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Pod struct {
	Name              string     `json:"name"`
	ID                string     `json:"id"`
	PriorityClassName string     `json:"priorityClassName"`
	WorkloadID        string     `json:"workloadId"`
	ClusterID         string     `json:"clusterId"`
	ProjectID         *string    `json:"projectId"`
	NodeName          *string    `json:"nodeName"`
	CreatedAt         time.Time  `json:"createdAt"`
	CompletedAt       *time.Time `json:"completedAt"`
	Containers        []struct {
		Name      string     `json:"name"`
		Image     string     `json:"image"`
		StartedAt *time.Time `json:"startedAt"`
	} `json:"containers"`
	CurrentNodePool    *string `json:"currentNodePool"`
	RequestedNodePool  *string `json:"requestedNodePool"`
	RequestedResources struct {
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
	} `json:"requestedResources"`
	AllocatedResources struct {
		GPU       float64 `json:"gpu"`
		GPUMemory string  `json:"gpuMemory"`
		CPU       float64 `json:"cpu"`
		CPUMemory string  `json:"cpuMemory"`
	} `json:"allocatedResources"`
	K8sPhase          string    `json:"k8sPhase"`
	TenantID          int       `json:"tenantId"`
	K8sPhaseUpdatedAt time.Time `json:"k8sPhaseUpdatedAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	ProjectName       string    `json:"projectName"`
	WorkloadName      string    `json:"workloadName"`
}

type ListPodsParams struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type ListPodsResponse struct {
	Pods []Pod `json:"pods"`
	Next int   `json:"next"`
}

// ListPods lists pods.
func (s *Service) ListPods(ctx context.Context, params ListPodsParams) (*ListPodsResponse, error) {
	if params.Limit > 500 {
		return nil, fmt.Errorf("limit must be less than 500")
	}

	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetQueryParams(map[string]string{
			"limit":     fmt.Sprintf("%d", params.Limit),
			"offset":    fmt.Sprintf("%d", params.Offset),
			"verbosity": "full",
			"completed": "false",
		}).
		SetResult(&ListPodsResponse{}).
		Get("/api/v1/workloads/pods")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to list pods, status code: %d", resp.StatusCode())
	}

	result := resp.Result().(*ListPodsResponse)

	j, err := json.Marshal(result)
	if err == nil {
		s.logger.Tracef("list pods response: %s", string(j))
	}

	return result, nil
}

// ListAllPods lists all pods.
func (s *Service) ListAllPods(ctx context.Context) ([]Pod, error) {
	pods := make([]Pod, 0)

	for {
		resp, err := s.ListPods(ctx, ListPodsParams{
			Limit:  500,
			Offset: len(pods),
		})
		if err != nil {
			return nil, err
		}

		pods = append(pods, resp.Pods...)

		if resp.Next == 0 {
			break
		}
	}

	return pods, nil
}
