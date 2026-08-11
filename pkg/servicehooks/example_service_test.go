package servicehooks_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/openmeterio/openmeter/pkg/servicehooks"
)

type lifecycleOperation string

const (
	lifecycleOperationCreate lifecycleOperation = "create"
	lifecycleOperationUpdate lifecycleOperation = "update"
	lifecycleOperationDelete lifecycleOperation = "delete"
)

type exampleResource struct {
	ID   string
	Name string
}

// lifecycleEvent is owned by the example service. The registry only preserves
// registration order and dispatches it; the service defines the meaning of
// Before and After.
type lifecycleEvent struct {
	Operation lifecycleOperation
	Before    *exampleResource
	After     *exampleResource
}

type exampleService struct {
	servicehooks.Registry[lifecycleEvent]

	resources map[string]exampleResource
}

func newExampleService() *exampleService {
	return &exampleService{
		resources: make(map[string]exampleResource),
	}
}

func (s *exampleService) Create(ctx context.Context, resource exampleResource) error {
	if _, exists := s.resources[resource.ID]; exists {
		return fmt.Errorf("resource %q already exists", resource.ID)
	}

	after := resource
	if err := s.Invoke(ctx, lifecycleEvent{
		Operation: lifecycleOperationCreate,
		After:     &after,
	}); err != nil {
		return fmt.Errorf("invoking create lifecycle hooks: %w", err)
	}

	s.resources[resource.ID] = resource

	return nil
}

func (s *exampleService) Update(ctx context.Context, resource exampleResource) error {
	current, exists := s.resources[resource.ID]
	if !exists {
		return fmt.Errorf("resource %q not found", resource.ID)
	}

	before := current
	after := resource
	if err := s.Invoke(ctx, lifecycleEvent{
		Operation: lifecycleOperationUpdate,
		Before:    &before,
		After:     &after,
	}); err != nil {
		return fmt.Errorf("invoking update lifecycle hooks: %w", err)
	}

	s.resources[resource.ID] = resource

	return nil
}

func (s *exampleService) Delete(ctx context.Context, id string) error {
	current, exists := s.resources[id]
	if !exists {
		return fmt.Errorf("resource %q not found", id)
	}

	before := current
	if err := s.Invoke(ctx, lifecycleEvent{
		Operation: lifecycleOperationDelete,
		Before:    &before,
	}); err != nil {
		return fmt.Errorf("invoking delete lifecycle hooks: %w", err)
	}

	delete(s.resources, id)

	return nil
}

func TestExampleServiceInvokesLifecycleHooksForEachOperation(t *testing.T) {
	service := newExampleService()
	var invocations []string
	createdResource := exampleResource{ID: "resource-1", Name: "initial"}
	updatedResource := exampleResource{ID: "resource-1", Name: "updated"}

	// Composition registers validation before observation because successful
	// Register calls define invocation order. Registry embedding promotes
	// Register directly onto the service.
	err := service.Register("validate-shape", servicehooks.HookFunc[lifecycleEvent](func(_ context.Context, event lifecycleEvent) error {
		switch event.Operation {
		case lifecycleOperationCreate:
			if event.Before != nil || event.After == nil {
				return errors.New("create requires only after state")
			}
			if *event.After != createdResource {
				return fmt.Errorf("unexpected create after state: %+v", *event.After)
			}
		case lifecycleOperationUpdate:
			if event.Before == nil || event.After == nil {
				return errors.New("update requires before and after state")
			}
			if *event.Before != createdResource || *event.After != updatedResource {
				return fmt.Errorf("unexpected update transition: before=%+v after=%+v", *event.Before, *event.After)
			}
		case lifecycleOperationDelete:
			if event.Before == nil || event.After != nil {
				return errors.New("delete requires only before state")
			}
			if *event.Before != updatedResource {
				return fmt.Errorf("unexpected delete before state: %+v", *event.Before)
			}
		default:
			return fmt.Errorf("unsupported lifecycle operation %q", event.Operation)
		}

		invocations = append(invocations, "validate:"+string(event.Operation))

		return nil
	}))
	if err != nil {
		t.Fatalf("registering validation hook: %v", err)
	}

	err = service.Register("observe", servicehooks.HookFunc[lifecycleEvent](func(_ context.Context, event lifecycleEvent) error {
		invocations = append(invocations, "observe:"+string(event.Operation))

		return nil
	}))
	if err != nil {
		t.Fatalf("registering observer hook: %v", err)
	}

	// Given a service whose hooks validate and observe every mutation.

	// When the resource is created, updated, and deleted.
	if err := service.Create(t.Context(), createdResource); err != nil {
		t.Fatalf("creating resource: %v", err)
	}
	if err := service.Update(t.Context(), updatedResource); err != nil {
		t.Fatalf("updating resource: %v", err)
	}
	if err := service.Delete(t.Context(), updatedResource.ID); err != nil {
		t.Fatalf("deleting resource: %v", err)
	}

	// Then every operation is delivered in registration order and the mutations
	// are committed only after their lifecycle hooks succeed.
	expectedInvocations := []string{
		"validate:create",
		"observe:create",
		"validate:update",
		"observe:update",
		"validate:delete",
		"observe:delete",
	}
	if len(invocations) != len(expectedInvocations) {
		t.Fatalf("unexpected invocation count: got %d, expected %d", len(invocations), len(expectedInvocations))
	}
	for i := range expectedInvocations {
		if invocations[i] != expectedInvocations[i] {
			t.Errorf("unexpected invocation at index %d: got %q, expected %q", i, invocations[i], expectedInvocations[i])
		}
	}
	if _, exists := service.resources[updatedResource.ID]; exists {
		t.Errorf("resource %q still exists after delete", updatedResource.ID)
	}
}

func TestExampleServiceDoesNotCommitMutationWhenLifecycleHookFails(t *testing.T) {
	service := newExampleService()
	rejected := errors.New("resource name is required")

	err := service.Register("require-name", servicehooks.HookFunc[lifecycleEvent](func(_ context.Context, event lifecycleEvent) error {
		if event.After != nil && event.After.Name == "" {
			return rejected
		}

		return nil
	}))
	if err != nil {
		t.Fatalf("registering validation hook: %v", err)
	}

	// Given a lifecycle hook which rejects an invalid candidate resource.
	resource := exampleResource{ID: "resource-1"}

	// When creation invokes the hook before committing the mutation.
	err = service.Create(t.Context(), resource)

	// Then the hook error is preserved and the resource is not stored.
	if !errors.Is(err, rejected) {
		t.Fatalf("expected lifecycle hook rejection, got %v", err)
	}
	if _, exists := service.resources[resource.ID]; exists {
		t.Errorf("resource %q was committed after lifecycle hook failure", resource.ID)
	}
}
