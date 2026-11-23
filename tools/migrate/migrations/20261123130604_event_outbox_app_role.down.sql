-- reverse: create index "eventoutbox_app_role_topic_transaction_id_id" to table: "event_outboxes"
DROP INDEX "eventoutbox_app_role_topic_transaction_id_id";
-- reverse: create index "eventoutbox_app_role_topic_id" to table: "event_outboxes"
DROP INDEX "eventoutbox_app_role_topic_id";
-- reverse: modify "event_outboxes" table
ALTER TABLE "event_outboxes" DROP COLUMN "app_role";
-- reverse: drop index "eventoutbox_topic_transaction_id_id" from table: "event_outboxes"
CREATE INDEX "eventoutbox_topic_transaction_id_id" ON "event_outboxes" ("topic", "transaction_id", "id");
-- reverse: drop index "eventoutbox_topic_id" from table: "event_outboxes"
CREATE INDEX "eventoutbox_topic_id" ON "event_outboxes" ("topic", "id");
