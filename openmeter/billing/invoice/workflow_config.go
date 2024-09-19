package invoice

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

type WorkflowConfig struct {
	ID         models.NamespacedID      `json:"id"`
	Collection WorkflowCollectionConfig `json:"collection"`
	Invoicing  WorkflowInvoicingConfig  `json:"invoicing"`
}

func (c *WorkflowConfig) Validate() error {
	if err := c.Collection.Validate(); err != nil {
		return err
	}

	if err := c.Invoicing.Validate(); err != nil {
		return err
	}

	return nil
}

type WorkflowCollectionConfig struct {
	AlignmentKind    billing.AlignmentKind `json:"alignmentKind"`
	CollectionPeriod time.Duration         `json:"collectionPeriod"`
}

func (c *WorkflowCollectionConfig) Validate() error {
	if c.AlignmentKind == "" {
		return ValidationError{
			Err: errors.New("alignment kind cannot be empty"),
		}
	}

	if c.CollectionPeriod < 0 {
		return ValidationError{
			Err: errors.New("collection period cannot be negative"),
		}
	}
	return nil
}

type WorkflowInvoicingConfig struct {
	AutoAdvance      *bool                    `json:"autoAdvance"`
	DraftPeriod      time.Duration            `json:"draftPeriod"`
	DueAfterDays     int                      `json:"dueAfterDays"`
	CollectionMethod billing.CollectionMethod `json:"collectionMethod"`
	Items            WorkflowItemsConfig      `json:"items"`
}

func (c *WorkflowInvoicingConfig) Validate() error {
	if c.DraftPeriod < 0 {
		return ValidationError{
			Err: errors.New("draft period cannot be negative"),
		}
	}

	if c.DueAfterDays < 0 {
		return ValidationError{
			Err: errors.New("due after days cannot be negative"),
		}
	}

	validCollectionMethods := billing.CollectionMethod("").Values()
	if !slices.Contains(validCollectionMethods, string(c.CollectionMethod)) {
		return ValidationError{
			Err: fmt.Errorf("collection method %s is not supported (valid methods: %s)", c.CollectionMethod, strings.Join(validCollectionMethods, ", ")),
		}
	}

	if err := c.Items.Validate(); err != nil {
		return err
	}

	return nil
}

type WorkflowItemsConfig struct {
	Resolution billing.GranualityResolution `json:"resolution"`
	PerSubject bool                         `json:"perSubject"`
}

func (c *WorkflowItemsConfig) Validate() error {
	validResolutions := billing.GranualityResolution("").Values()
	if !slices.Contains(validResolutions, string(c.Resolution)) {
		return ValidationError{
			Err: fmt.Errorf("resolution %s is not supported (valid resolutions: %s)", c.Resolution, strings.Join(validResolutions, ", ")),
		}
	}
	return nil
}
