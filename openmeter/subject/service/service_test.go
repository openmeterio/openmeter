package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjecttestutils "github.com/openmeterio/openmeter/openmeter/subject/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func Test_SubjectService(t *testing.T) {
	// Setup test environment
	env := subjecttestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := subjecttestutils.NewTestNamespace(t)

	t.Run("Create", func(t *testing.T) {
		sub1, err := env.SubjectService.Create(t.Context(), subject.CreateInput{
			Namespace:   namespace,
			Key:         "example-inc",
			DisplayName: lo.ToPtr("Example Inc."),
		})
		require.NoErrorf(t, err, "creating subject should not fail")
		require.NotEmpty(t, sub1, "subject must not be empty")

		sub2, err := env.SubjectService.Create(t.Context(), subject.CreateInput{
			Namespace:   namespace,
			Key:         "example-corp",
			DisplayName: lo.ToPtr("Example Corp."),
		})
		require.NoErrorf(t, err, "creating subject should not fail")
		require.NotEmpty(t, sub2, "subject must not be empty")

		sub3, err := env.SubjectService.Create(t.Context(), subject.CreateInput{
			Namespace:   namespace,
			Key:         sub2.Id,
			DisplayName: lo.ToPtr("Example LLC."),
		})
		require.NoErrorf(t, err, "creating subject should not fail")
		require.NotEmpty(t, sub3, "subject must not be empty")

		t.Run("Get", func(t *testing.T) {
			t.Run("ByID", func(t *testing.T) {
				byID, err := env.SubjectService.GetById(t.Context(), models.NamespacedID{
					Namespace: sub1.Namespace,
					ID:        sub1.Id,
				})
				require.NoErrorf(t, err, "getting subject by id should not fail")
				require.NotEmpty(t, sub1, "subject must not be empty")
				assert.Equalf(t, sub1.Id, byID.Id, "subject id must match")
			})

			t.Run("ByKey", func(t *testing.T) {
				byKey, err := env.SubjectService.GetByKey(t.Context(), models.NamespacedKey{
					Namespace: sub1.Namespace,
					Key:       sub1.Key,
				})
				require.NoErrorf(t, err, "getting subject by key should not fail")
				require.NotEmpty(t, sub1, "subject must not be empty")
				assert.Equalf(t, sub1.Id, byKey.Id, "subject id must match")
			})

			t.Run("ByIDOrKey", func(t *testing.T) {
				t.Run("ByID", func(t *testing.T) {
					byID, err := env.SubjectService.GetByIdOrKey(t.Context(), namespace, sub1.Id)
					require.NoErrorf(t, err, "getting subject by id should not fail")
					require.NotEmpty(t, sub1, "subject must not be empty")
					assert.Equalf(t, sub1.Id, byID.Id, "subject id must match")
				})

				t.Run("ByKey", func(t *testing.T) {
					byKey, err := env.SubjectService.GetByIdOrKey(t.Context(), namespace, sub3.Key)
					require.NoErrorf(t, err, "getting subject by key should not fail")
					require.NotEmpty(t, sub1, "subject must not be empty")
					assert.Equalf(t, sub2.Id, byKey.Id, "subject id must match")
				})
			})

			t.Run("List", func(t *testing.T) {
				subjects, err := env.SubjectService.List(t.Context(), namespace, subject.ListParams{
					Page: pagination.Page{
						PageSize:   100,
						PageNumber: 1,
					},
					SortBy: subject.ListSortByKeyAsc,
				})
				require.NoErrorf(t, err, "listing subjects should not fail")
				require.NotNilf(t, subjects, "subjects must not be nil")
				require.Lenf(t, subjects.Items, 3, "subjects must have 3 items")

				actualSubjectIDs := lo.Map(subjects.Items, func(item subject.Subject, index int) string {
					return item.Id
				})

				expectedSubjectIDs := []string{sub1.Id, sub2.Id, sub3.Id}

				assert.ElementsMatchf(t, actualSubjectIDs, expectedSubjectIDs, "subject ids must be in list")
			})

			t.Run("Update", func(t *testing.T) {
				displayName := "MegaCorp"
				stripeCustomerId := "cus_abc123"
				metadata := map[string]interface{}{
					"foo": "bar",
				}

				sub1, err = env.SubjectService.Update(t.Context(), subject.UpdateInput{
					ID:        sub1.Id,
					Namespace: sub1.Namespace,
					DisplayName: subject.OptionalNullable[string]{
						Value: lo.ToPtr(displayName),
						IsSet: true,
					},
					StripeCustomerId: subject.OptionalNullable[string]{
						Value: lo.ToPtr(stripeCustomerId),
						IsSet: true,
					},
					Metadata: subject.OptionalNullable[map[string]interface{}]{
						Value: lo.ToPtr(metadata),
						IsSet: true,
					},
				})
				require.NoErrorf(t, err, "updating subject should not fail")
				require.NotEmpty(t, sub1, "subjects must not be empty")
				assert.Equalf(t, displayName, lo.FromPtr(sub1.DisplayName), "subject display name must match")
				assert.Equalf(t, stripeCustomerId, lo.FromPtr(sub1.StripeCustomerId), "subject stripe customer id must match")
				assert.Equalf(t, metadata, sub1.Metadata, "subject metadata must match")
			})

			t.Run("Delete", func(t *testing.T) {
				t.Run("WithActiveEntitlement", func(t *testing.T) {
					cus, err := env.CustomerService.CreateCustomer(t.Context(), customer.CreateCustomerInput{
						Namespace: sub1.Namespace,
						CustomerMutate: customer.CustomerMutate{
							Key:  lo.ToPtr(sub1.Key),
							Name: lo.FromPtrOr(sub1.DisplayName, sub1.Key),
							UsageAttribution: &customer.CustomerUsageAttribution{
								SubjectKeys: []string{
									sub1.Key,
								},
							},
						},
					})
					require.NoErrorf(t, err, "creating customer should not fail")
					require.NotEmpty(t, cus, "customer must not be empty")
					require.ElementsMatchf(t, cus.UsageAttribution.SubjectKeys, []string{sub1.Key}, "customer usage attribution subject keys must match")

					feat, err := env.FeatureService.CreateFeature(t.Context(), feature.CreateFeatureInputs{
						Name:      "Feature 1",
						Key:       "feature-1",
						Namespace: namespace,
					})
					require.NoErrorf(t, err, "creating feature should not fail")
					require.NotEmpty(t, feat, "feature must not be empty")

					now := clock.Now().UTC()

					ent, err := env.EntitlementAdapter.CreateEntitlement(t.Context(), entitlement.CreateEntitlementRepoInputs{
						Namespace:  namespace,
						FeatureID:  feat.ID,
						FeatureKey: feat.Key,
						UsageAttribution: streaming.CustomerUsageAttribution{
							ID:  cus.ID,
							Key: cus.Key,
							SubjectKeys: []string{
								sub1.Key,
							},
						},
						EntitlementType:         entitlement.EntitlementTypeMetered,
						Metadata:                nil,
						ActiveFrom:              lo.ToPtr(now),
						ActiveTo:                nil,
						Annotations:             nil,
						MeasureUsageFrom:        lo.ToPtr(now),
						IssueAfterReset:         lo.ToPtr(1000.0),
						IssueAfterResetPriority: lo.ToPtr[uint8](1),
						IsSoftLimit:             lo.ToPtr(true),
						PreserveOverageAtReset:  lo.ToPtr(true),
					})
					require.NoErrorf(t, err, "creating entitlement should not fail")
					require.NotNilf(t, ent, "entitlement must not be nil")

					err = env.SubjectService.Delete(t.Context(), models.NamespacedID{
						Namespace: sub1.Namespace,
						ID:        sub1.Id,
					})
					require.NoErrorf(t, err, "We will not delete the entitlements as they belong to the customer not the subject")

					t.Run("Delete", func(t *testing.T) {
						at := now.Add(1 * time.Hour)

						err = env.EntitlementAdapter.DeactivateEntitlement(t.Context(), models.NamespacedID{
							Namespace: ent.Namespace,
							ID:        ent.ID,
						}, at)
						require.NoErrorf(t, err, "deactivating entitlement should not fail")

						clock.SetTime(at.Add(1 * time.Minute))
						defer clock.ResetTime()

						err = env.CustomerService.DeleteCustomer(t.Context(), customer.DeleteCustomerInput{
							Namespace: cus.Namespace,
							ID:        cus.ID,
						})
						require.NoErrorf(t, err, "deleting customer should not fail")

						err = env.SubjectService.Delete(t.Context(), models.NamespacedID{
							Namespace: sub1.Namespace,
							ID:        sub1.Id,
						})
						require.NoErrorf(t, err, "deleting subject should not fail")

						byID, err := env.SubjectService.GetById(t.Context(), models.NamespacedID{
							Namespace: sub1.Namespace,
							ID:        sub1.Id,
						})
						require.NoErrorf(t, err, "getting subject by id should not fail")
						require.NotEmpty(t, sub1, "subject must not be empty")
						assert.Equalf(t, sub1.Id, byID.Id, "subject id must match")
						assert.Truef(t, byID.IsDeleted(), "subject must be deleted")
					})
				})

				t.Run("WithNoActiveEntitlement", func(t *testing.T) {
					err = env.SubjectService.Delete(t.Context(), models.NamespacedID{
						Namespace: sub2.Namespace,
						ID:        sub2.Id,
					})
					require.NoErrorf(t, err, "deleting subject should not fail")

					t.Run("ReCreate", func(t *testing.T) {
						sub2, err = env.SubjectService.Create(t.Context(), subject.CreateInput{
							Namespace:   namespace,
							Key:         sub2.Key,
							DisplayName: sub2.DisplayName,
						})
						require.NoErrorf(t, err, "creating subject should not fail")
						require.NotEmpty(t, sub2, "subject must not be empty")
					})
				})
			})
		})
	})
}
