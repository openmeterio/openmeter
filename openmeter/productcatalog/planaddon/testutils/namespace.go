package testutils

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
)

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID
