package models

import (
	"fmt"
	"strings"
)

var _ FieldExpression = (*FieldValue)(nil)

type FieldValue struct {
	Field string
	Value string
}

func (f FieldValue) String() string {
	return f.Field + "='" + f.Value + "'"
}

func (f FieldValue) JSONPath() string {
	b := strings.Builder{}
	b.WriteString("@.")
	b.WriteString(f.Field)
	b.WriteString("=='")
	b.WriteString(f.Value)
	b.WriteString("'")

	return b.String()
}

type FieldExpression interface {
	fmt.Stringer

	JSONPath() string
}

var _ fmt.Stringer = (*FieldSelector)(nil)

type FieldSelector struct {
	field string
	exps  []FieldExpression
}

func (p FieldSelector) String() string {
	b := strings.Builder{}
	b.WriteString(p.field)

	if len(p.exps) > 0 {
		b.WriteString("[")

		for idx, exp := range p.exps {
			if idx > 0 {
				b.WriteString(", ")
			}

			b.WriteString(exp.String())
		}

		b.WriteString("]")
	}

	return b.String()
}

func (p FieldSelector) WithFilter(exps ...FieldExpression) FieldSelector {
	return FieldSelector{
		field: p.field,
		exps:  append(p.exps, exps...),
	}
}

func (p FieldSelector) JSONPath() string {
	b := strings.Builder{}
	b.WriteString(p.field)

	if len(p.exps) > 0 {
		b.WriteString("[?(")

		for idx, filter := range p.exps {
			if idx > 0 {
				b.WriteString(" && ")
			}

			b.WriteString(filter.JSONPath())
		}

		b.WriteString(")]")
	}

	return b.String()
}

func NewFieldSelector(field string, exps ...FieldExpression) FieldSelector {
	return FieldSelector{
		field: field,
		exps:  exps,
	}
}

var _ fmt.Stringer = (*FieldSelector)(nil)

type FieldSelectors []FieldSelector

func (s FieldSelectors) JSONPath() string {
	if len(s) == 0 {
		return ""
	}

	b := strings.Builder{}
	b.WriteString("$")

	for _, selector := range s {
		b.WriteString(".")
		b.WriteString(selector.JSONPath())
	}

	return b.String()
}

func (s FieldSelectors) String() string {
	if len(s) == 0 {
		return ""
	}

	b := strings.Builder{}
	for _, selector := range s {
		b.WriteString("/")
		b.WriteString(selector.String())
	}

	return b.String()
}

func (s FieldSelectors) WithPrefix(prefix FieldSelectors) FieldSelectors {
	return append(prefix, s...)
}
