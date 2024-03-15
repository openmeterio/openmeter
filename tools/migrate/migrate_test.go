// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migrate_test

import (
	"errors"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestUpDownUp(t *testing.T) {
	testDB := testutils.InitPostgresDB(t)
	defer testDB.PGDriver.Close()

	migrator, err := migrate.NewMigrate(testDB.URL, migrate.OMMigrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err1, err2 := migrator.Close()
		err := errors.Join(err1, err2)
		if err != nil {
			t.Fatal(err)
		}
	}()

	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}

	if err := migrator.Down(); err != nil {
		t.Fatal(err)
	}

	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
}
