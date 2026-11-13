//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=models/codegen.yaml -o models/models.gen.go ./openapi.yaml
//go:generate go run ./internal/cmd/modelaliases
//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ./openapi.yaml
package api
