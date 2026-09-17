package meta

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeUnsupported models.ErrorCode = "unsupported"

var ErrUnsupported = models.NewValidationIssue(
	ErrCodeUnsupported,
	"unsupported",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusInternalServerError),
)
