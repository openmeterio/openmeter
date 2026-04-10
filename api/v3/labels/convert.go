package labels

import (
	"encoding"
	"errors"
	"fmt"
	"strings"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/models"
)

const AnnotationsPrefix = "openmeter_"

type MetadataAnnotations struct {
	Metadata    models.Metadata
	Annotations models.Annotations
}

func ToMetadataAnnotations(labels *api.Labels) (MetadataAnnotations, error) {
	if labels == nil || len(*labels) == 0 {
		return MetadataAnnotations{}, nil
	}

	var (
		metadata    models.Metadata
		annotations models.Annotations
		errs        []error
	)

	for k, v := range *labels {
		if err := ValidateLabel(k, v); err != nil {
			errs = append(errs, err)

			continue
		}

		if strings.HasPrefix(k, AnnotationsPrefix) {
			if annotations == nil {
				annotations = make(models.Annotations)
			}

			annotations[strings.TrimPrefix(k, AnnotationsPrefix)] = v
		} else {
			if metadata == nil {
				metadata = make(models.Metadata)
			}

			metadata[k] = v
		}
	}

	return MetadataAnnotations{metadata, annotations}, models.NewNillableGenericValidationError(errors.Join(errs...))
}

func FromMetadataAnnotations(metadata models.Metadata, annotations models.Annotations) *api.Labels {
	labels := make(api.Labels, len(annotations)+len(metadata))

	for k, v := range metadata {
		if err := ValidateLabel(k, v); err != nil {
			continue
		}

		labels[k] = v
	}

	for k, v := range annotations {
		var val string

		switch vv := v.(type) {
		case fmt.Stringer:
			val = vv.String()
		case string:
			val = vv
		case encoding.TextMarshaler:
			b, err := vv.MarshalText()
			if err != nil {
				continue
			}

			val = string(b)
		}

		if err := ValidateLabel(k, val); err != nil {
			continue
		}

		if !strings.HasPrefix(k, AnnotationsPrefix) {
			k = AnnotationsPrefix + k
		}

		labels[k] = val
	}

	return &labels
}

func FromMetadata[T ~map[string]string](metadata T) *api.Labels {
	return FromMetadataAnnotations(models.Metadata(metadata), nil)
}

func ToMetadata(labels *api.Labels) (models.Metadata, error) {
	m, err := ToMetadataAnnotations(labels)

	return m.Metadata, err
}
