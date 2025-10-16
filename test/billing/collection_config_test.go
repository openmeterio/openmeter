package billing

import (
	"testing"

	"github.com/stretchr/testify/require"

	ombilling "github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestCollectionConfigValidate_Subscription_NoAnchoredDetailRequired(t *testing.T) {
	cfg := ombilling.CollectionConfig{
		Alignment: ombilling.AlignmentKindSubscription,
		Interval:  datetime.MustParseDuration(t, "PT1H"),
	}

	require.NoError(t, cfg.Validate())
}

func TestCollectionConfigValidate_Subscription_IgnoresAnchoredDetail(t *testing.T) {
	// Even if set by mistake, anchored detail should error because alignment is not anchored
	cfg := ombilling.CollectionConfig{
		Alignment:               ombilling.AlignmentKindSubscription,
		AnchoredAlignmentDetail: &ombilling.AnchoredAlignmentDetail{},
		Interval:                datetime.MustParseDuration(t, "PT1H"),
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "anchored alignment detail must be set when alignment is anchored")
}
