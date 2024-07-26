package ent1

//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --target ./db  --feature sql/upsert --feature sql/lock --feature sql/versioned-migration ./schema --template ../../expose.tpl --template ../../paginate.tpl
