//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=models/codegen.yaml ./openapi.yaml
//go:generate go run ../internal/cmd/modelaliases v3-server
//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ./openapi.yaml
package v3
