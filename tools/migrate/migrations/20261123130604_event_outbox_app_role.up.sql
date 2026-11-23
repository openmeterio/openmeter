-- drop index "eventoutbox_topic_id" from table: "event_outboxes"
DROP INDEX "eventoutbox_topic_id";
-- drop index "eventoutbox_topic_transaction_id_id" from table: "event_outboxes"
DROP INDEX "eventoutbox_topic_transaction_id_id";
-- modify "event_outboxes" table
ALTER TABLE "event_outboxes" ADD COLUMN "app_role" character varying NOT NULL DEFAULT '';
-- create index "eventoutbox_app_role_topic_id" to table: "event_outboxes"
CREATE INDEX "eventoutbox_app_role_topic_id" ON "event_outboxes" ("app_role", "topic", "id");
-- create index "eventoutbox_app_role_topic_transaction_id_id" to table: "event_outboxes"
CREATE INDEX "eventoutbox_app_role_topic_transaction_id_id" ON "event_outboxes" ("app_role", "topic", "transaction_id", "id");
