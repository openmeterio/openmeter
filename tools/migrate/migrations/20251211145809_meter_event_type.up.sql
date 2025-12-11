-- create index "meter_namespace_event_type" to table: "meters"
CREATE INDEX "meter_namespace_event_type" ON "meters" ("namespace", "event_type");
