package annotations

func Unset(a Annotation) Annotation {
	return Annotation{Key: a.Key, Value: ""}
}

func Annotate(val map[string]string, annotation Annotation) {
	if val == nil {
		val = make(map[string]string)
	}
	if annotation.Value == "" {
		delete(val, annotation.Key)
		return
	}
	val[annotation.Key] = annotation.Value
}
