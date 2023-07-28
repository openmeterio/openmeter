CREATE STREAM IF NOT EXISTS OM_{{ .Namespace | upper }}_DETECTED_EVENTS_STREAM
{{if eq .Format "JSON" }}
(
    key1 STRING KEY,
    key2 STRING KEY,
    id STRING,
    id_count BIGINT,
    type STRING,
    source STRING,
    subject STRING,
    time STRING,
    data STRING
)
{{end}}
WITH (
    KAFKA_TOPIC = {{ .Topic | squote }},
    KEY_FORMAT = {{ .Format | squote }},
    VALUE_FORMAT = {{ .Format | squote }}
);
