CREATE TABLE IF NOT EXISTS {{ .Database }}.{{ .EventsTableName }} (
    id String,
    type LowCardinality(String),
    subject String,
    source String,
    time DateTime,
    data String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(time)
ORDER BY (time, type, subject);

