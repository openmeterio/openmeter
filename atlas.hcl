env "local" {
  src = "${local.schema_src}"

  migration {
    dir = "${local.migrations_dir}"
    format = "${local.migrations_format}"
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }

  url = "${local.local_url}"

  // Define the URL of the Dev Database for this environment
  // See: https://atlasgo.io/concepts/dev-database
  dev = "docker://postgres/15/dev?search_path=public"

  lint {
    // Lint the effects of the 100 latest migration files
    latest = 100
  }
}

env "ci" {
  src = "${local.schema_src}"

  migration {
    dir = "${local.migrations_dir}"
    format = "${local.migrations_format}"
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }

  dev = "${local.ci_url}"
}

locals {
    // Define the directory where the schema definition resides.
    schema_src = "ent://openmeter/ent/schema"
    // Define the initial migration timestamp
    init_migration_ts = "20240826120919"
    // Define the directory where the migrations are stored.
    migrations_dir = "file://tools/migrate/migrations"
    // We use golang-migrate
    migrations_format = "golang-migrate"
    // Define common connection URLs
    local_url = "postgres://postgres:postgres@localhost:5432/postgres?search_path=public&sslmode=disable"
    ci_url = "postgres://postgres:postgres@postgres:5432/postgres?search_path=public&sslmode=disable"
}

lint {
    non_linear {
        error = true
    }

    destructive {
        error = false
    }

    data_depend {
        error = true
    }

    incompatible {
        error = true
    }
}
