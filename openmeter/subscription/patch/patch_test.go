package patch_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	psubs "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiplePatches(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	_, _ = getDefaultSpec(t, time.Now())

	t.Run("Can remove then add an item in a future phase", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("Can remove then add an item in the current phase", func(t *testing.T) {
		t.Skip("TODO")
	})
}

// utils
type testcase[T subscription.Applies] struct {
	Name            string
	Patch           T
	GetSpec         func(t *testing.T) *subscription.SubscriptionSpec
	Ctx             subscription.ApplyContext
	GetExpectedSpec func(t *testing.T) subscription.SubscriptionSpec
	ExpectedError   error
}

type testsuite[T subscription.Applies] struct {
	SystemTime time.Time
	TT         []testcase[T]
}

func (ts *testsuite[T]) Run(t *testing.T) {
	for _, tc := range ts.TT {
		t.Run(tc.Name, func(t *testing.T) {
			spec := tc.GetSpec(t)
			err := tc.Patch.ApplyTo(spec, tc.Ctx)

			if tc.ExpectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.True(t, errors.As(err, lo.ToPtr(tc.ExpectedError.(any))))
				assert.EqualError(t, err, tc.ExpectedError.Error())
			}

			if err == nil {
				require.NotNil(t, tc.GetExpectedSpec)
				expectedSpec := tc.GetExpectedSpec(t)
				if !reflect.DeepEqual(spec, &expectedSpec) {
					t.Errorf("expected: %+v, got: %+v", expectedSpec, spec)
				}
			}
		})
	}
}

func getDefaultSpec(t *testing.T, activeFrom time.Time) (*subscription.SubscriptionSpec, *psubs.Plan) {
	pInp := subscriptiontestutils.GetExamplePlanInput(t)
	p := &psubs.Plan{
		Plan: pInp.Plan,
		Ref:  &models.NamespacedID{Namespace: pInp.Namespace, ID: pInp.Key},
	}

	spec, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
		Name:       "Test Plan",
		CustomerId: "test_customer",
		Currency:   currencyx.Code("USD"),
		ActiveFrom: activeFrom,
	})
	require.Nil(t, err)

	return &spec, p
}
