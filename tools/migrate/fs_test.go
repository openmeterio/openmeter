package migrate

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/noignore
var noIgnoreFS embed.FS

//go:embed testdata/ignore
var ignoreFS embed.FS

func TestSourceWrapper(t *testing.T) {
	tests := []struct {
		Name string
		FS   fs.FS
		Path string

		ExpectedFiles []string
	}{
		{
			Name: "With ignore",
			FS:   ignoreFS,
			Path: "testdata/ignore",
			ExpectedFiles: []string{
				"20240826120919_init.down.sql",
				"20240826120919_init.up.sql",
				"20240917172257_billing-entities.down.sql",
				"20240917172257_billing-entities.up.sql",
			},
		},
		{
			Name: "Without ignore",
			FS:   noIgnoreFS,
			Path: "testdata/noignore",
			ExpectedFiles: []string{
				"20240826120919_init.down.sql",
				"20240826120919_init.up.sql",
				"20240903155435_entitlement-expired-index.down.sql",
				"20240903155435_entitlement-expired-index.up.sql",
				"20240917172257_billing-entities.down.sql",
				"20240917172257_billing-entities.up.sql",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			s := NewSourceWrapper(test.FS)

			entries, err := fs.ReadDir(s, test.Path)
			require.NoError(t, err, "failed to read dir", test.Path)

			assert.NotEmpty(t, entries, "expected at least one entry, got none")

			files := make([]string, 0, len(entries))
			for _, entry := range entries {
				files = append(files, entry.Name())
			}

			assert.ElementsMatch(t, files, test.ExpectedFiles, "mismatch in entries")
		})
	}
}
