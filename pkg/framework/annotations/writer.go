package annotations

import "github.com/openmeterio/openmeter/pkg/models"

func Unset(a Annotation) Annotation {
	return Annotation{Key: a.Key, Value: ""}
}

func Annotate(val *models.AnnotatedModel, annotation Annotation) {
	if val.Metadata == nil {
		val.Metadata = make(map[string]string)
	}
	if annotation.Value == "" {
		delete(val.Metadata, annotation.Key)
		return
	}
	val.Metadata[annotation.Key] = annotation.Value
}
