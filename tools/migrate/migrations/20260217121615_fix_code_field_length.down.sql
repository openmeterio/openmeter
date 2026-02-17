-- reverse: drop index "custom_currencies_code_key" from table: "custom_currencies"
CREATE UNIQUE INDEX "custom_currencies_code_key" ON "custom_currencies" ("code");
