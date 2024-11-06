-- reverse: create index "subscriptionpatchvalueremovephase_namespace_subscription_patch_" to table: "subscription_patch_value_remove_phases"
DROP INDEX "subscriptionpatchvalueremovephase_namespace_subscription_patch_";
-- reverse: create index "subscriptionpatchvalueremovephase_namespace_id" to table: "subscription_patch_value_remove_phases"
DROP INDEX "subscriptionpatchvalueremovephase_namespace_id";
-- reverse: create index "subscriptionpatchvalueremovephase_namespace" to table: "subscription_patch_value_remove_phases"
DROP INDEX "subscriptionpatchvalueremovephase_namespace";
-- reverse: create index "subscriptionpatchvalueremovephase_id" to table: "subscription_patch_value_remove_phases"
DROP INDEX "subscriptionpatchvalueremovephase_id";
-- reverse: create index "subscription_patch_value_remove_phases_subscription_patch_id_ke" to table: "subscription_patch_value_remove_phases"
DROP INDEX "subscription_patch_value_remove_phases_subscription_patch_id_ke";
-- reverse: create "subscription_patch_value_remove_phases" table
DROP TABLE "subscription_patch_value_remove_phases";
-- reverse: create index "subscriptionpatchvalueextendphase_namespace_subscription_patch_" to table: "subscription_patch_value_extend_phases"
DROP INDEX "subscriptionpatchvalueextendphase_namespace_subscription_patch_";
-- reverse: create index "subscriptionpatchvalueextendphase_namespace_id" to table: "subscription_patch_value_extend_phases"
DROP INDEX "subscriptionpatchvalueextendphase_namespace_id";
-- reverse: create index "subscriptionpatchvalueextendphase_namespace" to table: "subscription_patch_value_extend_phases"
DROP INDEX "subscriptionpatchvalueextendphase_namespace";
-- reverse: create index "subscriptionpatchvalueextendphase_id" to table: "subscription_patch_value_extend_phases"
DROP INDEX "subscriptionpatchvalueextendphase_id";
-- reverse: create index "subscription_patch_value_extend_phases_subscription_patch_id_ke" to table: "subscription_patch_value_extend_phases"
DROP INDEX "subscription_patch_value_extend_phases_subscription_patch_id_ke";
-- reverse: create "subscription_patch_value_extend_phases" table
DROP TABLE "subscription_patch_value_extend_phases";
-- reverse: create index "subscriptionpatchvalueaddphase_namespace_subscription_patch_id" to table: "subscription_patch_value_add_phases"
DROP INDEX "subscriptionpatchvalueaddphase_namespace_subscription_patch_id";
-- reverse: create index "subscriptionpatchvalueaddphase_namespace_id" to table: "subscription_patch_value_add_phases"
DROP INDEX "subscriptionpatchvalueaddphase_namespace_id";
-- reverse: create index "subscriptionpatchvalueaddphase_namespace" to table: "subscription_patch_value_add_phases"
DROP INDEX "subscriptionpatchvalueaddphase_namespace";
-- reverse: create index "subscriptionpatchvalueaddphase_id" to table: "subscription_patch_value_add_phases"
DROP INDEX "subscriptionpatchvalueaddphase_id";
-- reverse: create index "subscription_patch_value_add_phases_subscription_patch_id_key" to table: "subscription_patch_value_add_phases"
DROP INDEX "subscription_patch_value_add_phases_subscription_patch_id_key";
-- reverse: create "subscription_patch_value_add_phases" table
DROP TABLE "subscription_patch_value_add_phases";
-- reverse: create index "subscriptionpatchvalueadditem_namespace_subscription_patch_id" to table: "subscription_patch_value_add_items"
DROP INDEX "subscriptionpatchvalueadditem_namespace_subscription_patch_id";
-- reverse: create index "subscriptionpatchvalueadditem_namespace_id" to table: "subscription_patch_value_add_items"
DROP INDEX "subscriptionpatchvalueadditem_namespace_id";
-- reverse: create index "subscriptionpatchvalueadditem_namespace" to table: "subscription_patch_value_add_items"
DROP INDEX "subscriptionpatchvalueadditem_namespace";
-- reverse: create index "subscriptionpatchvalueadditem_id" to table: "subscription_patch_value_add_items"
DROP INDEX "subscriptionpatchvalueadditem_id";
-- reverse: create index "subscription_patch_value_add_items_subscription_patch_id_key" to table: "subscription_patch_value_add_items"
DROP INDEX "subscription_patch_value_add_items_subscription_patch_id_key";
-- reverse: create "subscription_patch_value_add_items" table
DROP TABLE "subscription_patch_value_add_items";
-- reverse: create index "subscriptionpatch_namespace_subscription_id" to table: "subscription_patches"
DROP INDEX "subscriptionpatch_namespace_subscription_id";
-- reverse: create index "subscriptionpatch_namespace_id" to table: "subscription_patches"
DROP INDEX "subscriptionpatch_namespace_id";
-- reverse: create index "subscriptionpatch_namespace" to table: "subscription_patches"
DROP INDEX "subscriptionpatch_namespace";
-- reverse: create index "subscriptionpatch_id" to table: "subscription_patches"
DROP INDEX "subscriptionpatch_id";
-- reverse: create "subscription_patches" table
DROP TABLE "subscription_patches";
-- reverse: create index "subscriptionentitlement_namespace_subscription_id_subscription_" to table: "subscription_entitlements"
DROP INDEX "subscriptionentitlement_namespace_subscription_id_subscription_";
-- reverse: create index "subscriptionentitlement_namespace_subscription_id" to table: "subscription_entitlements"
DROP INDEX "subscriptionentitlement_namespace_subscription_id";
-- reverse: create index "subscriptionentitlement_namespace_id" to table: "subscription_entitlements"
DROP INDEX "subscriptionentitlement_namespace_id";
-- reverse: create index "subscriptionentitlement_namespace_entitlement_id" to table: "subscription_entitlements"
DROP INDEX "subscriptionentitlement_namespace_entitlement_id";
-- reverse: create index "subscriptionentitlement_namespace" to table: "subscription_entitlements"
DROP INDEX "subscriptionentitlement_namespace";
-- reverse: create index "subscriptionentitlement_id" to table: "subscription_entitlements"
DROP INDEX "subscriptionentitlement_id";
-- reverse: create index "subscription_entitlements_entitlement_id_key" to table: "subscription_entitlements"
DROP INDEX "subscription_entitlements_entitlement_id_key";
-- reverse: create "subscription_entitlements" table
DROP TABLE "subscription_entitlements";
-- reverse: create index "price_namespace_subscription_id_key" to table: "prices"
DROP INDEX "price_namespace_subscription_id_key";
-- reverse: create index "price_namespace_subscription_id" to table: "prices"
DROP INDEX "price_namespace_subscription_id";
-- reverse: create index "price_namespace_id" to table: "prices"
DROP INDEX "price_namespace_id";
-- reverse: create index "price_namespace" to table: "prices"
DROP INDEX "price_namespace";
-- reverse: create index "price_id" to table: "prices"
DROP INDEX "price_id";
-- reverse: create "prices" table
DROP TABLE "prices";
-- reverse: create index "subscription_namespace_id" to table: "subscriptions"
DROP INDEX "subscription_namespace_id";
-- reverse: create index "subscription_namespace_customer_id" to table: "subscriptions"
DROP INDEX "subscription_namespace_customer_id";
-- reverse: create index "subscription_namespace" to table: "subscriptions"
DROP INDEX "subscription_namespace";
-- reverse: create index "subscription_id" to table: "subscriptions"
DROP INDEX "subscription_id";
-- reverse: create "subscriptions" table
DROP TABLE "subscriptions";
