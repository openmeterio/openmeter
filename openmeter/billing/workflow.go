package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/datetime"
)

type WorkflowConfig struct {
	Collection CollectionConfig  `json:"collection"`
	Invoicing  InvoicingConfig   `json:"invoicing"`
	Payment    PaymentConfig     `json:"payment"`
	Tax        WorkflowTaxConfig `json:"tax"`
}

func (c WorkflowConfig) Validate() error {
	if err := c.Collection.Validate(); err != nil {
		return fmt.Errorf("invalid collection config: %w", err)
	}

	if err := c.Invoicing.Validate(); err != nil {
		return fmt.Errorf("invalid invoice config: %w", err)
	}

	if err := c.Payment.Validate(); err != nil {
		return fmt.Errorf("invalid payment config: %w", err)
	}

	if err := c.Tax.Validate(); err != nil {
		return fmt.Errorf("invalid tax config: %w", err)
	}

	return nil
}

// CollectionConfig groups fields related to item collection.
type CollectionConfig struct {
	Alignment               AlignmentKind            `json:"alignment"`
	AnchoredAlignmentDetail *AnchoredAlignmentDetail `json:"anchoredAlignmentDetail,omitempty"`
	Interval                datetime.ISODuration     `json:"period,omitempty"`
}

func (c *CollectionConfig) Validate() error {
	if err := c.Alignment.Validate(); err != nil {
		return fmt.Errorf("invalid alignment: %s", c.Alignment)
	}

	if c.Alignment == AlignmentKindAnchored {
		if c.AnchoredAlignmentDetail == nil {
			return fmt.Errorf("anchored alignment detail must be set")
		}
	}

	if c.AnchoredAlignmentDetail != nil {
		if c.Alignment != AlignmentKindAnchored {
			return fmt.Errorf("anchored alignment detail must be set when alignment is anchored")
		}

		if err := c.AnchoredAlignmentDetail.Validate(); err != nil {
			return fmt.Errorf("invalid anchored alignment detail: %w", err)
		}
	}

	if !c.Interval.IsPositive() {
		return fmt.Errorf("item collection period must be greater or equal to 0")
	}

	return nil
}

// WorkflowTaxConfig groups fields related to tax settings.
type WorkflowTaxConfig struct {
	// Enable automatic tax calculation when tax is supported by the app.
	// For example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
	Enabled bool `json:"enabled"`

	// Enforce tax calculation when tax is supported by the app.
	// When enabled, OpenMeter will not allow to create an invoice without tax calculation.
	// Enforcement is different per apps, for example, Stripe app requires customer
	// to have a tax location when starting a paid subscription.
	Enforced bool `json:"enforced"`
}

// Validate validates the tax config.
func (c *WorkflowTaxConfig) Validate() error {
	if c.Enforced && !c.Enabled {
		return fmt.Errorf("tax is enforced but tax is not enabled")
	}

	return nil
}
