//go:build ignore
// +build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entcursor"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entexpose"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entmixinaccessor"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entpaginate"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entselectedparse"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entsetorclear"
	"github.com/openmeterio/openmeter/tools/migrate/viewgen"
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
			entcursor.New(),
			entexpose.New(),
			entmixinaccessor.New(),
			entpaginate.New(),
			entsetorclear.New(),
			entselectedparse.New(),
		),
	)
	if err != nil {
		log.Fatal("running ent codegen:", err)
	}

	if err := viewgen.GenerateFile("./schema", "../../tools/migrate/views.sql"); err != nil {
		log.Fatal("generating views SQL:", err)
	}
}
