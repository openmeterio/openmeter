package streaming

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateJSONPaths validates a meter's value-property and group-by paths against
// ClickHouse's parser, the authoritative grammar check. Both API versions call it.
func ValidateJSONPaths(ctx context.Context, connector Connector, valueProperty *string, groupBy map[string]string) error {
	if err := validateValuePropertyJSONPath(ctx, connector, valueProperty); err != nil {
		return err
	}

	return ValidateGroupByJSONPaths(ctx, connector, groupBy)
}

// validateValuePropertyJSONPath validates the value property; a nil one (count meter)
// is valid.
func validateValuePropertyJSONPath(ctx context.Context, connector Connector, valueProperty *string) error {
	if valueProperty == nil {
		return nil
	}

	isValid, err := connector.ValidateJSONPath(ctx, *valueProperty)
	if err != nil {
		return fmt.Errorf("validate json path in clickhouse: %w", err)
	}

	if !isValid {
		return models.NewGenericValidationError(fmt.Errorf("invalid meter value property JSONPath: %q", *valueProperty))
	}

	return nil
}

// ValidateGroupByJSONPaths validates group-by paths against ClickHouse. Update meters
// call it directly because the value property is immutable.
func ValidateGroupByJSONPaths(ctx context.Context, connector Connector, groupBy map[string]string) error {
	for groupByKey, jsonPath := range groupBy {
		isValid, err := connector.ValidateJSONPath(ctx, jsonPath)
		if err != nil {
			return fmt.Errorf("validate json path in clickhouse: %w", err)
		}

		if !isValid {
			return models.NewGenericValidationError(fmt.Errorf("invalid group by JSONPath %q for key %q", jsonPath, groupByKey))
		}
	}

	return nil
}
