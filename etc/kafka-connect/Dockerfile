FROM confluentinc/cp-kafka-connect-base:7.4.1

RUN wget https://github.com/ClickHouse/clickhouse-kafka-connect/releases/download/v0.0.19/clickhouse-kafka-connect-v0.0.19.zip
RUN set -xe && \
    confluent-hub install --no-prompt confluentinc/kafka-connect-jdbc:10.7.2 && \
    confluent-hub install --no-prompt ./clickhouse-kafka-connect-v0.0.19.zip
