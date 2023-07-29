CREATE TABLE IF NOT EXISTS OM_{{ .Namespace | upper }}_DETECTED_EVENTS
WITH (
    KAFKA_TOPIC = {{ .Topic | squote }},
    KEY_FORMAT = {{ .Format | squote }},
    VALUE_FORMAT = {{ .Format | squote }},
    PARTITIONS = {{ .Partitions }}
) AS
SELECT
    `id` AS `key1`,
    `source` AS `key2`,
    AS_VALUE(`id`) AS `id`,
    EARLIEST_BY_OFFSET(`type`) AS `type`,
    AS_VALUE(`source`) AS `source`,
    EARLIEST_BY_OFFSET(`subject`) AS `subject`,
    EARLIEST_BY_OFFSET(`time`) AS `time`,
    EARLIEST_BY_OFFSET(`data`) AS `data`,
    COUNT(`id`) as `id_count`
FROM OM_{{ .Namespace | upper }}_EVENTS
WINDOW TUMBLING (
    SIZE {{ .Retention }} DAYS,
    RETENTION {{ .Retention }} DAYS
)
GROUP BY
    `id`,
    `source`;
