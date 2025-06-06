package subject

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Subject represents a subject in the system.
type Subject struct {
	Id          string
	Key         string
	DisplayName *string
	Metadata    map[string]interface{}
	// Use customer application entity instead
	StripeCustomerId *string
}

func (s Subject) Validate() error {
	var errs []error

	if s.Key == "" {
		errs = append(errs, errors.New("key is required"))
	}

	return errors.Join(errs...)
}

// ToAPIModel converts the subject to the API model.
func (s Subject) ToAPIModel() api.Subject {
	var metadata *map[string]interface{}

	if s.Metadata != nil {
		m := map[string]interface{}{}

		for k, v := range s.Metadata {
			m[k] = v
		}

		metadata = &m
	}

	return api.Subject{
		Id:               s.Id,
		Key:              s.Key,
		DisplayName:      s.DisplayName,
		Metadata:         metadata,
		StripeCustomerId: s.StripeCustomerId,
	}
}

// SubjectKey is key only version of Subject
// Used in in entitlements events to reduce payload size
type SubjectKey struct {
	Key string `json:"key"`
}

func (s SubjectKey) Validate() error {
	if s.Key == "" {
		return errors.New("key is required")
	}

	return nil
}

// ListAll returns all the subjects.
// Helper function for listing all customers. Page param will be ignored.
func ListAll(ctx context.Context, service Service, namespace string, params ListParams) ([]Subject, error) {
	subjects := []Subject{}
	limit := 100
	page := 1

	for {
		params := params
		params.Page = pagination.NewPage(page, limit)

		result, err := service.List(ctx, namespace, params)
		if err != nil {
			return nil, fmt.Errorf("failed to list all subjects: %w", err)
		}

		for _, subject := range result.Items {
			if subject == nil {
				return nil, fmt.Errorf("subject is nil")
			}

			subjects = append(subjects, *subject)
		}

		if len(result.Items) < limit {
			break
		}

		page++
	}

	return subjects, nil
}
