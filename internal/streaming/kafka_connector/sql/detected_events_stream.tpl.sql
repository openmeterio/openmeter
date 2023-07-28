CREATE STREAM IF NOT EXISTS OM_{{ .Namespace | upper }}_DETECTED_EVENTS_STREAM
WITH (
    KAFKA_TOPIC = {{ .Topic | squote }},
    KEY_FORMAT = 'JSON_SR',
    VALUE_FORMAT = 'JSON_SR'
);
