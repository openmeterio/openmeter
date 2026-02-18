package adapter

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func mapProRatingToDB(proRating productcatalog.ProRatingConfig) charges.ProRatingModeAdapterEnum {
	if proRating.Enabled && proRating.Mode == productcatalog.ProRatingModeProratePrices {
		return charges.ProRatingAdapterModeEnumProratePrices
	}

	return charges.ProRatingAdapterModeEnumNoProrate
}

func mapProRatingFromDB(proRating charges.ProRatingModeAdapterEnum) (productcatalog.ProRatingConfig, error) {
	switch proRating {
	case charges.ProRatingAdapterModeEnumProratePrices:
		return productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}, nil
	case charges.ProRatingAdapterModeEnumNoProrate:
		return productcatalog.ProRatingConfig{
			Enabled: false,
		}, nil
	default:
		return productcatalog.ProRatingConfig{}, fmt.Errorf("invalid pro rating mode %s", proRating)
	}
}
