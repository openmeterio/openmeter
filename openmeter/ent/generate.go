package entdb

//go:generate go run -mod=readonly entc.go
//go:generate go run -mod=readonly ../../tools/migrate/cmd/viewgen -schema ./schema -out ../../tools/migrate/views.sql
