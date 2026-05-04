CREATE TABLE IF NOT EXISTS distributed_locks (
    "name" varchar(255) NOT NULL,
    "record_version_number" int8 NULL,
    "data" bytea NULL,
    "owner" varchar(255) NULL,
    CONSTRAINT distributed_locks_pkey PRIMARY KEY ("name")
);

CREATE SEQUENCE IF NOT EXISTS distributed_locks_rvn CYCLE OWNED BY distributed_locks.record_version_number;
