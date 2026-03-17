package v3

// GetSpecBytes returns the raw embedded OpenAPI specification bytes.
// Used by libopenapi for request/response validation.
func GetSpecBytes() ([]byte, error) {
	return rawSpec()
}
