//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --config codegen.yaml openapi.yaml
package kafkaconnect

import (
	_ "github.com/deepmap/oapi-codegen/pkg/runtime"
)
