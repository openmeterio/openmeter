package entitlement_test

import (
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSchedulingConstraint(t *testing.T) {
	tt := []struct {
		name     string
		ents     []entitlement.Entitlement
		expected error
	}{
		{
			name:     "Should work on empty input",
			ents:     []entitlement.Entitlement{},
			expected: nil,
		},
		{
			name: "Should error if entitlements belong to different features",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-01T00:00:00Z",
				}),
				getEnt(t, getEntInp{
					feature:    "feature2",
					customerID: "subject1",
					createdAt:  "2021-01-01T00:00:00Z",
				}),
			},
			expected: fmt.Errorf("entitlements must belong to the same feature, found [feature1 feature2]"),
		},
		{
			name: "Should error if entitlements belong to different subjects",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-01T00:00:00Z",
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject2",
					createdAt:  "2021-01-01T00:00:00Z",
				}),
			},
			expected: fmt.Errorf("entitlements must belong to the same subject, found [subject1 subject2]"),
		},
		{
			name: "Should not error for single entitlement",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-01T00:00:00Z",
				}),
			},
			expected: nil,
		},
		{
			name: "Should not error for non overlapping entitlements",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-01T00:00:00Z",
					activeTo:   lo.ToPtr("2021-01-02T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					deletedAt:  lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					activeFrom: lo.ToPtr("2021-01-03T00:00:00Z"),
					deletedAt:  lo.ToPtr("2021-01-04T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					activeFrom: lo.ToPtr("2021-01-04T00:00:00Z"),
					activeTo:   lo.ToPtr("2021-01-05T00:00:00Z"),
					deletedAt:  lo.ToPtr("2021-01-06T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					activeFrom: lo.ToPtr("2021-01-05T00:00:00Z"),
					activeTo:   lo.ToPtr("2021-01-06T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-07T00:00:00Z",
				}),
			},
			expected: nil,
		},
		{
			name: "Should error for overlapping entitlements if one is indefinite",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					id:         "1",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-01T00:00:00Z",
				}),
				getEnt(t, getEntInp{
					id:         "2",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					deletedAt:  lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
			},
			expected: fmt.Errorf("constraint violated: 1 is active at the same time as 2"),
		},
		{
			name: "Should error for overlapping entitlements",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					id:         "5",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-01T00:00:00Z",
					activeTo:   lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					id:         "2",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					deletedAt:  lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
			},
			expected: fmt.Errorf("constraint violated: 5 is active at the same time as 2"),
		},
		{
			name: "Should not error for zero length cadence with overlapping start",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					id:         "1",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2024-01-01T00:00:00Z",
					activeTo:   lo.ToPtr("2024-01-01T00:00:00Z"), // Zero-length
				}),
				getEnt(t, getEntInp{
					id:         "2",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2024-01-01T00:00:00Z",
					activeTo:   lo.ToPtr("2024-01-02T00:00:00Z"),
				}),
			},
			expected: nil,
		},
		{
			name: "Should not error for multiple zero length cadences at the same time",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					id:         "1",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2024-01-01T00:00:00Z",
					activeTo:   lo.ToPtr("2024-01-01T00:00:00Z"), // Zero-length
				}),
				getEnt(t, getEntInp{
					id:         "2",
					feature:    "feature1",
					customerID: "subject1",
					createdAt:  "2024-01-01T00:00:00Z",
					activeTo:   lo.ToPtr("2024-01-01T00:00:00Z"), // Zero-length
				}),
			},
			expected: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := entitlement.ValidateUniqueConstraint(tc.ents)
			if tc.expected == nil {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, tc.expected.Error())
			}
		})
	}
}

type getEntInp struct {
	id         string
	feature    string
	customerID string
	createdAt  string
	activeFrom *string
	activeTo   *string
	deletedAt  *string
}

func getEnt(t *testing.T, inp getEntInp) entitlement.Entitlement {
	createdAt := testutils.GetRFC3339Time(t, inp.createdAt)

	ent := entitlement.Entitlement{
		GenericProperties: entitlement.GenericProperties{
			EntitlementType: entitlement.EntitlementTypeBoolean,
			FeatureKey:      inp.feature,
			CustomerID:      inp.customerID,
			ManagedModel: models.ManagedModel{
				CreatedAt: createdAt,
			},
		},
	}

	if inp.activeFrom != nil {
		ent.ActiveFrom = lo.ToPtr(testutils.GetRFC3339Time(t, *inp.activeFrom))
	}

	if inp.activeTo != nil {
		ent.ActiveTo = lo.ToPtr(testutils.GetRFC3339Time(t, *inp.activeTo))
	}

	if inp.deletedAt != nil {
		ent.DeletedAt = lo.ToPtr(testutils.GetRFC3339Time(t, *inp.deletedAt))
	}

	ent.ID = inp.id

	return ent
}
