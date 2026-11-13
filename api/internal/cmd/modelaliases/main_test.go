package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloudAliasRejectsSharedOnlyJSONMethod(t *testing.T) {
	const model = "package models\ntype Invoice struct{ ID string }\n"
	server, err := declarations("server.go", []byte(model+`func (i Invoice) MarshalJSON() ([]byte, error) {
		return []byte("null"), nil
	}`))
	require.NoError(t, err)
	client, err := declarations("client.go", []byte(model))
	require.NoError(t, err)

	_, _, err = mergeModels(server, client)
	require.ErrorContains(t, err, "Invoice differs from shared models")
}

func TestMergeCloudModels(t *testing.T) {
	server, err := declarations("server.go", []byte(`package models
type Shared string
type ServerOnly int
`))
	require.NoError(t, err)
	cloud, err := declarations("cloud.go", []byte(`package models
import "time"
type Shared string
type State string
const Ready State = "ready"
type CloudOnly struct { Shared Shared; Since time.Time; State State }
func (s State) Valid() bool { return s == Ready }
type newAuthContextKey string
const NewAuthScopes newAuthContextKey = "NewAuth.Scopes"
`))
	require.NoError(t, err)

	extra, client, err := mergeModels(server, cloud)
	require.NoError(t, err)
	require.Contains(t, string(extra), `"time"`)
	require.Contains(t, string(extra), "type CloudOnly struct")
	require.Contains(t, string(extra), "func (s State) Valid() bool")
	require.NotContains(t, string(extra), "type Shared ")
	require.NotContains(t, string(extra), "NewAuthScopes")

	pub, err := aliases("openmeter", legacyModelsImport, client)
	require.NoError(t, err)
	for _, name := range []string{"Shared", "ServerOnly", "State", "CloudOnly"} {
		require.Contains(t, string(pub), "type "+name+" = models."+name)
	}

	require.Contains(t, string(pub), "const Ready = models.Ready")
	require.Contains(t, string(pub), `const NewAuthScopes newAuthContextKey = "NewAuth.Scopes"`)

	pub, err = aliases("api", legacyModelsImport, server)
	require.NoError(t, err)
	require.NotContains(t, string(pub), "CloudOnly")
}
