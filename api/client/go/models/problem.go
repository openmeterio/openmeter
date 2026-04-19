package models

import "fmt"

// ProblemType contains a URI that identifies the problem type.
type ProblemType string

const ProblemTypeDefault = ProblemType("about:blank")

// StatusProblem is the RFC 7807 problem detail object.
type StatusProblem struct {
	Err error `json:"-"`

	Type       ProblemType            `json:"type"`
	Title      string                 `json:"title"`
	Status     int                    `json:"status"`
	Detail     string                 `json:"detail,omitempty"`
	Instance   string                 `json:"instance,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (p *StatusProblem) Error() string {
	if p.Err == nil {
		return fmt.Sprintf("[%s] %s", p.Title, p.Detail)
	}

	return fmt.Sprintf("[%s] %s - %s", p.Title, p.Err.Error(), p.Detail)
}
