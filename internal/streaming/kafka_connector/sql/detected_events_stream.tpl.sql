CREATE STREAM IF NOT EXISTS OM_DETECTED_EVENTS_STREAM
WITH (
    KAFKA_TOPIC = {{ .Topic | squote }},
    KEY_FORMAT = 'JSON_SR',
    VALUE_FORMAT = 'JSON_SR'
);
