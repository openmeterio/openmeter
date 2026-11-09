package historical

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestWriterBuildStampsCopiedAnnotations(t *testing.T) {
	original := models.Annotations{
		ledger.AnnotationChargeID:        "charge",
		ledger.AnnotationWriterCommitSHA: "previous writer",
		ledger.AnnotationWriterVersion:   "previous release",
	}

	build := WriterBuild{CommitSHA: "current writer", Version: "unknown"}
	require.Equal(t, models.Annotations{
		ledger.AnnotationChargeID:        "charge",
		ledger.AnnotationWriterCommitSHA: "current writer",
	}, build.stamp(original))
	require.Equal(t, "previous writer", original[ledger.AnnotationWriterCommitSHA])
	require.Equal(t, "previous release", original[ledger.AnnotationWriterVersion])
	require.Nil(t, (WriterBuild{}).stamp(nil))
}
