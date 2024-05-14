package models

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ProblemExtraFields struct {
	InstanceIDs []string `json:"instance_ids"`
	Valid       bool     `json:"bool"`
}

func TestProblemParse(t *testing.T) {
	problem := NewStatusProblemWithExtension(
		context.Background(),
		errors.New("fail"),
		http.StatusConflict,
		ProblemExtraFields{
			InstanceIDs: []string{"1", "2", "3"},
			Valid:       false,
		},
	)

	marshaled, err := json.Marshal(&problem)
	assert.NoError(t, err)

	unmarshaledProblem := StatusProblemWithExtension[ProblemExtraFields]{}
	assert.NoError(t, json.Unmarshal(marshaled, &unmarshaledProblem))

	// Skip non-json marshaled fields
	problem.Problem.Err = nil
	assert.Equal(t, problem, unmarshaledProblem)
}
