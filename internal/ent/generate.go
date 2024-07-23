package entcodegen

//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --target ./db  --feature sql/upsert --feature sql/lock --feature sql/versioned-migration --template ../../pkg/framework/entutils/expose.tpl ./schema
