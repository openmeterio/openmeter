package billing

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ValidationIssueSeverity string

const (
	ValidationIssueSeverityCritical ValidationIssueSeverity = "critical"
	ValidationIssueSeverityWarning  ValidationIssueSeverity = "warning"

	ValidationComponentOpenMeter         ComponentName = "openmeter"
	ValidationComponentOpenMeterMetering ComponentName = "openmeter.metering"
	ValidationComponentProductCatalog    ComponentName = "product_catalog"

	ValidationIssueCodeLineEngineCollectionCompletedFailed = "line_engine_collection_completed_failed"
)

func (ValidationIssueSeverity) Values() []string {
	return []string{
		string(ValidationIssueSeverityCritical),
		string(ValidationIssueSeverityWarning),
	}
}

type ValidationIssue struct {
	ID        string     `json:"id,omitempty"`
	CreatedAt time.Time  `json:"createdAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Severity  ValidationIssueSeverity `json:"severity"`
	Message   string                  `json:"message"`
	Code      string                  `json:"code,omitempty"`
	Component ComponentName           `json:"component,omitempty"`
	// TODO: migrate billing validation issues to models.ValidationIssue's FieldDescriptor-based path.
	// Deprecated: this field should be moved to models.ValidationIssue's FieldDescriptor-based path implementation.
	Path       string             `json:"path,omitempty"`
	Attributes models.Annotations `json:"attributes,omitempty"`
}

func (i ValidationIssue) EncodeAsErrorExtension() map[string]interface{} {
	out := map[string]interface{}{
		"severity": i.Severity,
		"message":  i.Message,
	}

	if i.Component != "" {
		out["component"] = i.Component
	}

	if i.Path != "" {
		out["path"] = i.Path
	}

	if i.Code != "" {
		out["code"] = i.Code
	}

	if len(i.Attributes) > 0 {
		out["attributes"] = i.Attributes
	}

	return out
}

func (i ValidationIssue) Clone() (ValidationIssue, error) {
	clone := i

	attributes, err := i.Attributes.Clone()
	if err != nil {
		return ValidationIssue{}, fmt.Errorf("cloning attributes: %w", err)
	}

	clone.Attributes = attributes

	return clone, nil
}

func (i ValidationIssue) Error() string {
	return i.Message
}

// Is identifies coded validation issues independently of their contextual details.
func (i ValidationIssue) Is(target error) bool {
	other, ok := target.(ValidationIssue)

	return ok && i.Code != "" && i.Code == other.Code
}

func NewValidationWarning(code, message string) ValidationIssue {
	return ValidationIssue{
		Severity: ValidationIssueSeverityWarning,
		Message:  message,
		Code:     code,
	}
}

func NewValidationError(code, message string) ValidationIssue {
	return ValidationIssue{
		Severity: ValidationIssueSeverityCritical,
		Message:  message,
		Code:     code,
	}
}

type ComponentName string

func AppTypeCapabilityToComponent(appType app.AppType, cap app.CapabilityType, op string) ComponentName {
	return ComponentName(fmt.Sprintf("app.%s.%s.%s", appType, cap, op))
}

type componentWrapper struct {
	component ComponentName
	err       error
}

func (c componentWrapper) Error() string {
	return string(c.component) + ": " + c.err.Error()
}

func (c componentWrapper) Unwrap() error {
	return c.err
}

// ValidationWithComponent wraps an error with a component name, if error is nil, it returns nil.
// This can be used to add context to an error when we are crossing service boundaries. When
// wrappers are nested, the outermost service boundary defines the extracted issue's component.
// Component context does not change whether an error is a validation issue.
func ValidationWithComponent(component ComponentName, err error) error {
	if err == nil {
		return nil
	}

	return componentWrapper{
		component: component,
		err:       err,
	}
}

type messageWrapper struct {
	prefix string
	err    error
}

func (m messageWrapper) Error() string {
	return m.prefix + ": " + m.err.Error()
}

func (m messageWrapper) Unwrap() error {
	return m.err
}

// ValidationWithMessage adds human-readable guidance to an error and optional structured context.
// Attribute maps are merged from left to right, with later values taking precedence. Message context
// does not change whether an error is a validation issue. When the wrapped error contains validation
// issues, the guidance and attributes are added to each extracted issue.
func ValidationWithMessage(err error, prefix string, attributes ...models.Attributes) error {
	if err == nil {
		return nil
	}

	wrapped := error(messageWrapper{
		prefix: prefix,
		err:    err,
	})

	mergedAttributes := models.Attributes{}
	for _, attributeSet := range attributes {
		mergedAttributes = mergedAttributes.Merge(attributeSet)
	}

	if len(mergedAttributes) == 0 {
		return wrapped
	}

	return ValidationWithAttributes(models.Annotations(mergedAttributes.AsStringMap()), wrapped)
}

type fieldPrefixWrapper struct {
	prefix string
	err    error
}

func (f fieldPrefixWrapper) Error() string {
	return f.prefix + ": " + f.err.Error()
}

func (f fieldPrefixWrapper) Unwrap() error {
	return f.err
}

// ValidationWithFieldPrefix wraps an error with a field prefix, if error is nil, it returns nil
// This can be used to delegate validation duties to a sub-entity. (e.g. lines don't need to know about
// the path in the invoice they are residing at)
// Field context does not change whether an error is a validation issue.
func ValidationWithFieldPrefix(prefix string, err error) error {
	if err == nil {
		return nil
	}

	return fieldPrefixWrapper{
		prefix: prefix,
		err:    err,
	}
}

type attributesWrapper struct {
	attributes models.Annotations
	err        error
}

func (a attributesWrapper) Error() string {
	if len(a.attributes) == 0 {
		return a.err.Error()
	}

	keys := make([]string, 0, len(a.attributes))
	for key := range a.attributes {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	attributes := make([]string, 0, len(keys))
	for _, key := range keys {
		attributes = append(attributes, fmt.Sprintf("%s=%v", key, a.attributes[key]))
	}

	return fmt.Sprintf("%s [%s]", a.err.Error(), strings.Join(attributes, ","))
}

func (a attributesWrapper) Unwrap() error {
	return a.err
}

// ValidationWithAttributes adds contextual attributes to validation issues extracted from err.
// When wrappers are nested, attributes from the outermost wrapper take precedence.
// Attribute context does not change whether an error is a validation issue.
func ValidationWithAttributes(attributes models.Annotations, err error) error {
	if err == nil {
		return nil
	}

	return attributesWrapper{
		attributes: attributes,
		err:        err,
	}
}

type asValidationIssueWrapper struct {
	err      error
	severity ValidationIssueSeverity
}

func (w asValidationIssueWrapper) Error() string {
	return w.err.Error()
}

func (w asValidationIssueWrapper) Unwrap() error {
	return w.err
}

type WrapAsValidationIssueOption interface {
	apply(*asValidationIssueWrapper)
}

type wrapAsValidationIssueOptionFunc func(*asValidationIssueWrapper)

func (f wrapAsValidationIssueOptionFunc) apply(wrapper *asValidationIssueWrapper) {
	f(wrapper)
}

func WithWarningSeverity() WrapAsValidationIssueOption {
	return wrapAsValidationIssueOptionFunc(func(wrapper *asValidationIssueWrapper) {
		wrapper.severity = ValidationIssueSeverityWarning
	})
}

// WrapAsValidationIssue explicitly converts ordinary leaf errors in err into validation issues
// when they are extracted. Converted errors are critical by default. Existing validation issues
// retain their severity and metadata. When conversion wrappers are nested, the outermost wrapper
// controls the severity of ordinary errors.
func WrapAsValidationIssue(err error, options ...WrapAsValidationIssueOption) error {
	if err == nil {
		return nil
	}

	wrapper := asValidationIssueWrapper{
		err:      err,
		severity: ValidationIssueSeverityCritical,
	}
	for _, option := range options {
		option.apply(&wrapper)
	}

	return wrapper
}

type ValidationIssues []ValidationIssue

func (v ValidationIssues) HasComponent(component ComponentName) bool {
	return slices.ContainsFunc(v, func(issue ValidationIssue) bool {
		return issue.Component == component
	})
}

func (v ValidationIssues) WithoutComponent(component ComponentName) ValidationIssues {
	issues := slices.DeleteFunc(slices.Clone(v), func(issue ValidationIssue) bool {
		return issue.Component == component
	})

	return issues
}

// ToValidationIssues extracts validation issues from an error tree. If the error is nil, it returns nil.
// If any leaf error is neither a billing or models ValidationIssue nor explicitly wrapped by
// WrapAsValidationIssue, it returns the original error tree. This behavior allows critical system
// errors to remain distinct from validation issues.
func ToValidationIssues(errIn error) (ValidationIssues, error) {
	if errIn == nil {
		return nil, nil
	}

	issues, err := toValidationIssue(errIn, "", "", "", nil, "")
	if err != nil {
		return nil, errIn
	}

	return issues, nil
}

// IsValidationIssueOnly reports whether the error tree contains validation
// issues only and no system errors.
func IsValidationIssueOnly(err error) bool {
	if err == nil {
		return false
	}

	_, systemErr := ToValidationIssues(err)

	return systemErr == nil
}

func (v ValidationIssues) RemoveMetaForCompare() ValidationIssues {
	return lo.Map(v, func(issue ValidationIssue, _ int) ValidationIssue {
		issue.CreatedAt = time.Time{}
		issue.UpdatedAt = time.Time{}
		issue.DeletedAt = nil
		issue.ID = ""

		return issue
	})
}

func (v ValidationIssues) Clone() (ValidationIssues, error) {
	if v == nil {
		return nil, nil
	}

	return lo.MapErr(v, func(issue ValidationIssue, _ int) (ValidationIssue, error) {
		return issue.Clone()
	})
}

func (v ValidationIssues) AsError() error {
	if len(v) == 0 {
		return nil
	}

	return errors.Join(lo.Map(v, func(issue ValidationIssue, _ int) error {
		return issue
	})...)
}

func (v ValidationIssues) Map(f func(ValidationIssue, int) ValidationIssue) ValidationIssues {
	return lo.Map(v, f)
}

type errorsUnwrap interface {
	Unwrap() []error
}

type errorUnwrap interface {
	Unwrap() error
}

func addStartingSlashIfNeeded(path string) string {
	if path == "" {
		return ""
	}

	if path[0] == '/' {
		return path
	}

	return "/" + path
}

func appendToPrefix(prefix string, field string) string {
	if prefix == "" {
		return addStartingSlashIfNeeded(field)
	}

	if field == "" {
		return addStartingSlashIfNeeded(prefix)
	}

	return addStartingSlashIfNeeded(prefix + "/" + field)
}

func appendMessagePrefix(prefix string, message string) string {
	if prefix == "" {
		return message
	}

	return prefix + ": " + message
}

func toValidationIssue(err error, fieldPrefix string, component ComponentName, messagePrefix string, attributes models.Annotations, ordinaryErrorSeverity ValidationIssueSeverity) ([]ValidationIssue, error) {
	if err == nil {
		return nil, nil
	}

	// let's see if the current error requires special handling (as switch's
	// ordering is non-deterministic, we first have a typeswitch for the special cases)
	switch errT := err.(type) {
	case asValidationIssueWrapper:
		severity := ordinaryErrorSeverity
		if severity == "" {
			severity = errT.severity
		}

		return toValidationIssue(errT.err, fieldPrefix, component, messagePrefix, attributes, severity)
	case componentWrapper:
		issueComponent := component
		if issueComponent == "" {
			issueComponent = errT.component
		}

		return toValidationIssue(errT.err, fieldPrefix, issueComponent, messagePrefix, attributes, ordinaryErrorSeverity)
	case fieldPrefixWrapper:
		return toValidationIssue(errT.err, appendToPrefix(fieldPrefix, errT.prefix), component, messagePrefix, attributes, ordinaryErrorSeverity)
	case messageWrapper:
		return toValidationIssue(errT.err, fieldPrefix, component, appendMessagePrefix(messagePrefix, errT.prefix), attributes, ordinaryErrorSeverity)
	case attributesWrapper:
		mergedAttributes, err := errT.attributes.Merge(attributes)
		if err != nil {
			return nil, fmt.Errorf("merging validation issue attributes: %w", err)
		}

		return toValidationIssue(errT.err, fieldPrefix, component, messagePrefix, mergedAttributes, ordinaryErrorSeverity)
	case ValidationIssue:
		issueComponent := component
		if issueComponent == "" {
			issueComponent = errT.Component
		}

		mergedAttributes, err := errT.Attributes.Merge(attributes)
		if err != nil {
			return nil, fmt.Errorf("merging validation issue attributes: %w", err)
		}

		mergedAttributes, err = mergedAttributes.Clone()
		if err != nil {
			return nil, fmt.Errorf("cloning validation issue attributes: %w", err)
		}

		return []ValidationIssue{
			{
				Severity:   errT.Severity,
				Message:    appendMessagePrefix(messagePrefix, errT.Message),
				Code:       errT.Code,
				Path:       appendToPrefix(fieldPrefix, errT.Path),
				Component:  issueComponent,
				Attributes: mergedAttributes,
			},
		}, nil
	case models.ValidationIssue:
		return modelValidationIssueAsBillingValidationIssue(errT, fieldPrefix, component, messagePrefix, attributes)
	case *models.ValidationIssue:
		if errT == nil {
			return nil, nil
		}

		return modelValidationIssueAsBillingValidationIssue(*errT, fieldPrefix, component, messagePrefix, attributes)
	}

	switch errT := err.(type) {
	case errorsUnwrap:
		var issues []ValidationIssue
		for _, e := range errT.Unwrap() {
			out, err := toValidationIssue(e, fieldPrefix, component, messagePrefix, attributes, ordinaryErrorSeverity)
			if err != nil {
				return nil, err
			}
			if len(out) > 0 {
				issues = append(issues, out...)
			}
		}

		return issues, nil
	case errorUnwrap:
		return toValidationIssue(errT.Unwrap(), fieldPrefix, component, messagePrefix, attributes, ordinaryErrorSeverity)
	default:
		if ordinaryErrorSeverity != "" {
			issueAttributes, cloneErr := attributes.Clone()
			if cloneErr != nil {
				return nil, fmt.Errorf("cloning validation issue attributes: %w", cloneErr)
			}

			return []ValidationIssue{
				{
					Severity:   ordinaryErrorSeverity,
					Message:    appendMessagePrefix(messagePrefix, err.Error()),
					Path:       fieldPrefix,
					Component:  component,
					Attributes: issueAttributes,
				},
			}, nil
		} else {
			return nil, err
		}
	}
}

func modelValidationIssueAsBillingValidationIssue(issue models.ValidationIssue, fieldPrefix string, component ComponentName, messagePrefix string, attributes models.Annotations) ([]ValidationIssue, error) {
	issueComponent := component
	if issueComponent == "" {
		issueComponent = ComponentName(issue.Component())
	}

	issueAttributes := models.Annotations(issue.Attributes().AsStringMap())
	mergedAttributes, err := issueAttributes.Merge(attributes)
	if err != nil {
		return nil, fmt.Errorf("merging validation issue attributes: %w", err)
	}

	mergedAttributes, err = mergedAttributes.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning validation issue attributes: %w", err)
	}

	// TODO: Migrate billing validation issues to models.ValidationIssue so FieldDescriptor can be
	// preserved without lossy conversion to the legacy string path.
	return []ValidationIssue{
		{
			Severity:   ValidationIssueSeverity(issue.Severity().String()),
			Message:    appendMessagePrefix(messagePrefix, issue.Message()),
			Code:       string(issue.Code()),
			Path:       appendToPrefix(fieldPrefix, ""),
			Component:  issueComponent,
			Attributes: mergedAttributes,
		},
	}, nil
}

type ValidationIssueRecorder struct {
	issues []error
}

type ValidationIssueRecorderOption interface {
	apply(*validationIssueRecorderOptions)
}

type validationIssueRecorderOptions struct {
	attributes models.Annotations
	component  ComponentName
	path       string
}

func (o validationIssueRecorderOptions) wrap(err error) error {
	if len(o.attributes) > 0 {
		err = ValidationWithAttributes(o.attributes, err)
	}

	if o.component != "" {
		err = ValidationWithComponent(o.component, err)
	}

	if o.path != "" {
		err = ValidationWithFieldPrefix(o.path, err)
	}

	return err
}

type validationIssueRecorderOptionFunc func(*validationIssueRecorderOptions)

func (f validationIssueRecorderOptionFunc) apply(options *validationIssueRecorderOptions) {
	f(options)
}

// WithAttributes adds attributes to errors handled by ValidationIssueRecorder.
func WithAttributes(attributes models.Annotations) ValidationIssueRecorderOption {
	return validationIssueRecorderOptionFunc(func(options *validationIssueRecorderOptions) {
		options.attributes = attributes
	})
}

// WithComponent assigns a component to errors handled by ValidationIssueRecorder.
func WithComponent(component ComponentName) ValidationIssueRecorderOption {
	return validationIssueRecorderOptionFunc(func(options *validationIssueRecorderOptions) {
		options.component = component
	})
}

// WithPath prefixes the path of errors handled by ValidationIssueRecorder.
func WithPath(path string) ValidationIssueRecorderOption {
	return validationIssueRecorderOptionFunc(func(options *validationIssueRecorderOptions) {
		options.path = path
	})
}

// Record adds an error to the recorder if it doesn't contain any system error. If a system
// error is encountered, err is returned with the supplied context.
//
// Should be used in the following pattern:
//
//	_, err := calculateDetailedLines(stdLine)
//	if err := validationRecorder.Record(
//		err,
//		WithAttributes(models.Annotations{AttributeKeyLineID: stdLine.ID}),
//		WithComponent(ValidationComponentOpenMeter),
//		WithPath("lines/"+stdLine.ID),
//	); err != nil {
//		return nil, fmt.Errorf("calculating detailed lines for line[%s]: %w", stdLine.ID, err)
//	}
func (r *ValidationIssueRecorder) Record(err error, options ...ValidationIssueRecorderOption) error {
	if err == nil {
		return nil
	}

	var appliedOptions validationIssueRecorderOptions
	for _, option := range options {
		option.apply(&appliedOptions)
	}
	err = appliedOptions.wrap(err)

	if !IsValidationIssueOnly(err) {
		return err
	}

	// At this point we know that the errors are all validation issues
	r.issues = append(r.issues, err)
	return nil
}

// ErrorsOrNil returns the recorded validation issues as a single error. If there are no issues, it returns nil.
func (r *ValidationIssueRecorder) ErrorsOrNil() error {
	return errors.Join(r.issues...)
}
