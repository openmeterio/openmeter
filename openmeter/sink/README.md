# Usage sink

## When a batch notification is published

The sink publishes `EventBatchedIngest` after the storage write returns successfully.
The notification’s `storedAt` is the time the sink prepared the notification.

With `aggregation.asyncInsert=true` and `aggregation.asyncInsertWait=false`, ClickHouse
acknowledges a batch when it enters the insert buffer. A consumer that queries usage on
receipt can therefore see the previous total. Flushing that buffer later does not emit a
second notification. If consumers need to query the inserted data on receipt, use
synchronous inserts or set `aggregation.asyncInsertWait=true` when using async inserts.

The sink’s [ClickHouse writer](../../app/common/streaming.go) reads these settings from
`aggregation`. The similarly named `sink.storage` fields have no effect on this writer.
