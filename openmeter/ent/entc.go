//go:build ignore
// +build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entexpose"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entpaginate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entselectedparse"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entsetorclear"
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
			Package: "github.com/openmeterio/openmeter/openmeter/ent/db",
		},
		entc.Extensions(
			entexpose.New(),
			entpaginate.New(),
			entsetorclear.New(),
			entselectedparse.New(),
		),
	)
	if err != nil {
		log.Fatal("running ent codegen:", err)
	}
}
