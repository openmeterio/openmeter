package legacylineage

import (
	"encoding/json"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
)

const annotationCorrectionSegments = "ledger.correction.legacy_segments"

// CorrectionAnnotations carries the ledger planner's exact selections through
// realization persistence. The compatibility writer must not select amounts again.
func CorrectionAnnotations(selected map[string]alpacadecimal.Decimal) (models.Annotations, error) {
	encoded, err := json.Marshal(selected)
	if err != nil {
		return nil, err
	}
	return models.Annotations{annotationCorrectionSegments: string(encoded)}, nil
}

func CorrectionSelections(annotations models.Annotations) (map[string]alpacadecimal.Decimal, error) {
	value, exists := annotations[annotationCorrectionSegments]
	if !exists {
		return nil, nil
	} // No selection is needed when the realization has no legacy root.
	encoded, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid legacy correction selections")
	}
	var out map[string]alpacadecimal.Decimal
	if err := json.Unmarshal([]byte(encoded), &out); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("legacy correction selections must be present")
	}
	for id, amount := range out {
		if id == "" || !amount.IsPositive() {
			return nil, fmt.Errorf("invalid legacy correction selection %s", id)
		}
	}
	return out, nil
}
