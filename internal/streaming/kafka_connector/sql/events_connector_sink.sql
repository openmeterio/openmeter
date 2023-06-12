CREATE SINK CONNECTOR SINK_EVENTS_PG WITH (
    'connector.class'                         = 'io.confluent.connect.jdbc.JdbcSinkConnector',
    'connection.url'                          = 'jdbc:postgresql://postgres:5432/postgres',
    'connection.user'                         = 'postgres',
    'connection.password'                     = 'postgres',
    'topics'                                  = 'om_events',
    'key.converter'                           = 'io.confluent.connect.json.JsonSchemaConverter',
    'key.converter.schema.registry.url'       = 'http://schema:8081',
    'value.converter'                         = 'io.confluent.connect.json.JsonSchemaConverter',
    'value.converter.schema.registry.url'     = 'http://schema:8081',
    'auto.create'                             = 'true',
    'auto.evolve'                             = 'true',
    'delete.enabled'                          = 'false',
    'insert.mode'                             = 'insert',
    'pk.mode'                                 = 'kafka'
);