FROM confluentinc/cp-kafka-connect-base:7.4.1

RUN set -xe && \
    confluent-hub install --no-prompt confluentinc/kafka-connect-jdbc:10.7.2 && \
    confluent-hub install --no-prompt clickhouse/clickhouse-kafka-connect:0.0.17
