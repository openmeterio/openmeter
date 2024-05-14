package models

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/jsonx"
)

// StatusProblemWithExtension is the RFC 7807 response body with additional fields from ExtensionMemberT
// added to the same root object as Problem. Warning: ExtensionMemberT must be a struct
type StatusProblemWithExtension[ExtensionMemberT any] struct {
	Problem          *StatusProblem
	ExtensionMembers ExtensionMemberT
}

func NewStatusProblemWithExtension[ExtensionMemberT any](ctx context.Context, err error, status int, ext ExtensionMemberT) *StatusProblemWithExtension[ExtensionMemberT] {
	return &StatusProblemWithExtension[ExtensionMemberT]{
		Problem:          newStatusProblem(ctx, err, status),
		ExtensionMembers: ext,
	}
}

func (p *StatusProblemWithExtension[ExtensionMemberT]) MarshalJSON() ([]byte, error) {
	probMarshaled, err := json.Marshal(p.Problem)
	if err != nil {
		return nil, err
	}

	extMarshaled, err := json.Marshal(p.ExtensionMembers)
	if err != nil {
		return nil, err
	}

	// extMarshaled is the base, so that problem takes precedence in case of any conflicts
	return jsonx.Merge(extMarshaled, probMarshaled)
}

func (p *StatusProblemWithExtension[ExtensionMemberT]) UnmarshalJSON(jsonBytes []byte) error {
	if p.Problem == nil {
		p.Problem = &StatusProblem{}
	}

	if err := json.Unmarshal(jsonBytes, &p.Problem); err != nil {
		return err
	}

	if err := json.Unmarshal(jsonBytes, &p.ExtensionMembers); err != nil {
		return err
	}
	return nil
}

func (p *StatusProblemWithExtension[ExtensionMemberT]) Respond(w http.ResponseWriter, r *http.Request) {
	// Respond
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(p)

	w.Header().Set("Content-Type", ProblemContentType)
	w.WriteHeader(p.Problem.Status)
	_, _ = w.Write(buf.Bytes())
}

func (p *StatusProblemWithExtension[ExtensionMemberT]) Error() string {
	return p.Problem.Error()
}

func (p *StatusProblemWithExtension[ExtensionMemberT]) ProblemType() ProblemType {
	return p.Problem.ProblemType()
}

func (p *StatusProblemWithExtension[ExtensionMemberT]) ProblemTitle() string {
	return p.Problem.ProblemTitle()
}
