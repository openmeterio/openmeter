-- Postgres extensions required by the Ent schema. Atlas does not manage
-- extensions, so they must exist before the schema DDL is applied.
-- gin_trgm_ops: customer_subjects.subject_key trigram index.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
