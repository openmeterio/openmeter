env "local" {
  // Declare where the schema definition resides.
  src = "ent://internal/ent/schema"

  migration {
    // Define the directory where the migrations are stored.
    dir = "file://migrations"
  }

  // Define the URL of the database which is managed in this environment.
  url = "postgres://postgres:postgres@localhost:5432/postgres?search_path=public&sslmode=disable"

  // Define the URL of the Dev Database for this environment
  // See: https://atlasgo.io/concepts/dev-database
  dev = "docker://postgres/15/dev?search_path=public"

  lint {
    // Lint the effects of the 100 latest migration files
    latest = 100
  }
}

// CAN be used for all remote deployments
env "remote" {
  // Declare where the schema definition resides.
  src = "ent://internal/ent/schema"

  migration {
    // Define the directory where the migrations are stored.
    dir = "file://migrations"
    // Remote deployments already had auto deploy present
    baseline = "20240806133826"
  }

  // Define the URL of the Dev Database for this environment
  // See: https://atlasgo.io/concepts/dev-database
  dev = "docker://postgres/15/dev?search_path=public"
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
