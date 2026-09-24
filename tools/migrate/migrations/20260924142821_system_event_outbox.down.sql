-- reverse: create index "eventoutbox_topic_id" to table: "event_outboxes"
DROP INDEX "eventoutbox_topic_id";
-- reverse: create "event_outboxes" table
DROP TABLE "event_outboxes";
