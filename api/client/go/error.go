package openmeter

// ErrResponse renderer type for handling all sorts of errors.
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err error `json:"-"` // low-level runtime error

	StatusCode int    `json:"statusCode"`        // http response status code
	StatusText string `json:"status"`            // user-level status message
	AppCode    int64  `json:"code,omitempty"`    // application-specific error code
	Message    string `json:"message,omitempty"` // application-level error message, for debugging
}
