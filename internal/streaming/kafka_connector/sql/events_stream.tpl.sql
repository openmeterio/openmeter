CREATE STREAM IF NOT EXISTS OM_EVENTS
WITH (
    KAFKA_TOPIC = {{ .Topic | squote }},
    KEY_FORMAT = 'JSON_SR',
    VALUE_FORMAT = 'JSON_SR',
    PARTITIONS = {{ .Partitions }},
    KEY_SCHEMA_ID = {{ .KeySchemaId }},
    VALUE_SCHEMA_ID = {{ .ValueSchemaId }}
);
