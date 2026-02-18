//go:build ignore
// +build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entcursor"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entexpose"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entpaginate"
)

func main() {
	err := entc.Generate("./schema",
		&gen.Config{
			Features: []gen.Feature{
				gen.FeatureVersionedMigration,
				gen.FeatureLock,
				gen.FeatureUpsert,
				gen.FeatureExecQuery,
			},
			Target:  "./db",
			Schema:  "./schema",
			Package: "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db",
		},
		entc.Extensions(entcursor.New(), entexpose.New(), entpaginate.New()),
	)
	if err != nil {
		log.Fatal("running ent codegen:", err)
	}
}
