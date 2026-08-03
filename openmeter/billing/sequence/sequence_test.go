package sequence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommitModeValidate(t *testing.T) {
	tests := []struct {
		name    string
		mode    CommitMode
		wantErr string
	}{
		{
			name: "with caller",
			mode: CommitModeWithCaller,
		},
		{
			name: "independent",
			mode: CommitModeIndependent,
		},
		{
			name:    "missing",
			wantErr: "commit mode is required",
		},
		{
			name:    "invalid",
			mode:    CommitMode("invalid"),
			wantErr: "commit mode is invalid: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mode.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestDefinitionValidateRequiresCommitMode(t *testing.T) {
	def := Definition{
		Prefix:         "INV",
		SuffixTemplate: "{{.NextSequenceNumber}}",
		Scope:          "invoices/test",
	}

	err := def.Validate()
	require.EqualError(t, err, "commit mode is required")
}

func TestDefinitionValidateRejectsInvalidCommitMode(t *testing.T) {
	def := Definition{
		Prefix:         "INV",
		SuffixTemplate: "{{.NextSequenceNumber}}",
		Scope:          "invoices/test",
		CommitMode:     CommitMode("invalid"),
	}

	err := def.Validate()
	require.EqualError(t, err, "commit mode is invalid: invalid")
}
