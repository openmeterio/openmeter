-- reverse: create index "addonratecard_namespace_key_deleted_at" to table: "addon_rate_cards"
DROP INDEX "addonratecard_namespace_key_deleted_at";
-- reverse: create index "addonratecard_namespace_id" to table: "addon_rate_cards"
DROP INDEX "addonratecard_namespace_id";
-- reverse: create index "addonratecard_namespace" to table: "addon_rate_cards"
DROP INDEX "addonratecard_namespace";
-- reverse: create index "addonratecard_id" to table: "addon_rate_cards"
DROP INDEX "addonratecard_id";
-- reverse: create index "addonratecard_addon_id_key" to table: "addon_rate_cards"
DROP INDEX "addonratecard_addon_id_key";
-- reverse: create index "addonratecard_addon_id_feature_key" to table: "addon_rate_cards"
DROP INDEX "addonratecard_addon_id_feature_key";
-- reverse: create "addon_rate_cards" table
DROP TABLE "addon_rate_cards";
-- reverse: create index "addon_namespace_key_version" to table: "addons"
DROP INDEX "addon_namespace_key_version";
-- reverse: create index "addon_namespace_key_deleted_at" to table: "addons"
DROP INDEX "addon_namespace_key_deleted_at";
-- reverse: create index "addon_namespace_id" to table: "addons"
DROP INDEX "addon_namespace_id";
-- reverse: create index "addon_namespace" to table: "addons"
DROP INDEX "addon_namespace";
-- reverse: create index "addon_id" to table: "addons"
DROP INDEX "addon_id";
-- reverse: create index "addon_annotations" to table: "addons"
DROP INDEX "addon_annotations";
-- reverse: create "addons" table
DROP TABLE "addons";
