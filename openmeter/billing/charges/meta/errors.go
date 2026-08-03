package meta

import (
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ErrCustomCurrencyNotSupported is returned when a custom currency is not supported yet, use this as a
// marker to signify where we need to continue adding support for custom currencies.
var ErrCustomCurrencyNotSupported = errors.New("custom currency is not supported")

const ErrCodeUnsupported models.ErrorCode = "unsupported"

var ErrUnsupported = models.NewValidationIssue(
	ErrCodeUnsupported,
	"unsupported",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusInternalServerError),
)
