package events

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventParsingAndValidation(t *testing.T) {
	snapshot := EntitlementsBalanceSnapshotV1Properties{}
	ev, err := CreateEntitlementsBalanceSnapshotMessage(EventMeta{
		Type:   "v1.entitlements.balance.snapshot",
		ID:     "123",
		Source: "test",
		Time:   time.Now(),
	}, snapshot)

	assert.NoError(t, err)
	assert.NotNil(t, ev)

	schema, err := SchemaValidator()
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	bytea := ev.Data()
	assert.NotNil(t, bytea)
	fmt.Println(string(bytea))

	assert.NoError(t, schema.Validate(bytea))
}
