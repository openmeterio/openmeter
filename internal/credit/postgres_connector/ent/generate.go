package ent

//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --target ./db  --feature sql/upsert --feature sql/versioned-migration ./schema
