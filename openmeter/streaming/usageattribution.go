package streaming

import (
	"errors"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Customer is a customer that can be used in a meter query
type Customer interface {
	GetUsageAttribution() CustomerUsageAttribution
}

// NewCustomerUsageAttribution creates a new CustomerUsageAttribution
func NewCustomerUsageAttribution(id string, key *string, subjectKeys []string) CustomerUsageAttribution {
	customerUsageAttribution := CustomerUsageAttribution{
		ID:          id,
		Key:         key,
		SubjectKeys: subjectKeys,
	}

	if customerUsageAttribution.SubjectKeys == nil {
		customerUsageAttribution.SubjectKeys = []string{}
	}

	return customerUsageAttribution
}

// CustomerUsageAttribution holds customer fields that map usage to a customer
type CustomerUsageAttribution struct {
	// We don't attribute usage to the customer by ID but we need it to be able to map subjects to customers
	ID string `json:"id"`
	// We attribute usage to the customer by key
	Key *string `json:"key"`
	// We attribute usage to the customer by subject keys
	SubjectKeys []string `json:"subjectKeys"`
}

// Validate validates the CustomerUsageAttribution
func (ua CustomerUsageAttribution) Validate() error {
	if ua.ID == "" {
		return models.NewGenericValidationError(errors.New("usage attribution must have an id"))
	}

	if ua.Key == nil && len(ua.SubjectKeys) == 0 {
		return models.NewGenericValidationError(errors.New("usage attribution must have a key or subject keys"))
	}

	for _, subjectKey := range ua.SubjectKeys {
		if subjectKey == "" {
			return models.NewGenericValidationError(errors.New("subject key cannot be empty"))
		}
	}

	return nil
}

// GetValues returns the values by which the usage is attributed to the customer
func (ua CustomerUsageAttribution) GetValues() []string {
	attributions := []string{}

	if ua.Key != nil {
		attributions = append(attributions, *ua.Key)
	}

	attributions = append(attributions, ua.SubjectKeys...)

	return attributions
}

// Equal checks if two CustomerUsageAttributions are equal
func (ua CustomerUsageAttribution) Equal(other CustomerUsageAttribution) bool {
	if ua.ID != other.ID {
		return false
	}

	// Compare Key values, handling nil cases
	if (ua.Key == nil) != (other.Key == nil) {
		return false
	}
	if ua.Key != nil && *ua.Key != *other.Key {
		return false
	}

	return slices.Equal(ua.SubjectKeys, other.SubjectKeys)
}
