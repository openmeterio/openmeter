CREATE TABLE IF NOT EXISTS om_events (
    namespace String,
    id String,
    type LowCardinality(String),
    subject String,
    source String,
    time DateTime,
    data String,
    ingested_at DateTime,
    stored_at DateTime,
    INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4,
    store_row_id String
) ENGINE = MergeTree
PARTITION BY toYYYYMM(time)
ORDER BY (namespace, type, subject, toStartOfHour(time))
