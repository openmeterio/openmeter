package app

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var TestType = appentitybase.AppTypeStripe

type AppHandlerTestSuite struct {
	Env TestEnv

	namespace string
}

// setupNamespace can be used to set up an independent namespace for testing, it contains a single
// feature and rule with a channel. For more complex scenarios, additional setup might be required.
func (s *AppHandlerTestSuite) setupNamespace(t *testing.T) {
	t.Helper()

	s.namespace = ulid.Make().String()
}

// TestGet tests the getting of a app by ID
func (s *AppHandlerTestSuite) TestGetMarketplaceListing(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.App()

	// Listing
	expectedListing := appstripeentity.StripeMarketplaceListing

	require.NotNil(t, expectedListing, "Expected Listing must not be nil")

	// Get the listing
	registryItem, err := service.Get(ctx, appentity.MarketplaceGetInput{
		Type: TestType,
	})

	listing := registryItem.Listing

	require.NoError(t, err, "Fetching listing must not return error")
	require.NotNil(t, listing, "Listing must not be nil")
	require.Equal(t, TestType, listing.Type, "Listing type must match")
	require.Equal(t, expectedListing.Name, listing.Name, "Listing name must match")
	require.Equal(t, expectedListing.Description, listing.Description, "Listing description must match")
	require.Equal(t, expectedListing.IconURL, listing.IconURL, "Listing icon url must match")
	require.ElementsMatch(t, expectedListing.Capabilities, listing.Capabilities, "Listing capabilities must match")
}

// TestList tests the listing of apps
func (s *AppHandlerTestSuite) TestListMarketplaceListings(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.App()

	// Get the listing
	list, err := service.List(ctx, appentity.MarketplaceListInput{
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
	})

	require.NoError(t, err, "Fetching listings must not return error")
	require.NotNil(t, list, "Listings must not be nil")
	require.Equal(t, 1, list.TotalCount, "Listings total count must be 1")
	require.Equal(t, 1, list.Page.PageNumber, "Listings page must be 0")
	require.Len(t, list.Items, 1, "Listings must have a single item")
	require.Equal(t, list.Items[0].Listing, appstripeentity.StripeMarketplaceListing, "Listings must match")
}
