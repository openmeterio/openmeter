package taxcode

const (
	// AnnotationKeyManagedBy indicates what created/owns this tax code.
	AnnotationKeyManagedBy = "managed_by"

	// AnnotationValueManagedBySystem is set when the tax code was auto-created
	// by the system (e.g. via GetOrCreateByAppMapping).
	AnnotationValueManagedBySystem = "system"
)
