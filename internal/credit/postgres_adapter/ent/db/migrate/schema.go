// Code generated by ent, DO NOT EDIT.

package migrate

import (
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// BalanceSnapshotsColumns holds the columns for the "balance_snapshots" table.
	BalanceSnapshotsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeInt, Increment: true},
		{Name: "namespace", Type: field.TypeString},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "updated_at", Type: field.TypeTime},
		{Name: "deleted_at", Type: field.TypeTime, Nullable: true},
		{Name: "owner_id", Type: field.TypeString, SchemaType: map[string]string{"postgres": "char(26)"}},
		{Name: "grant_balances", Type: field.TypeJSON, SchemaType: map[string]string{"postgres": "jsonb"}},
		{Name: "balance", Type: field.TypeFloat64, SchemaType: map[string]string{"postgres": "numeric"}},
		{Name: "overage", Type: field.TypeFloat64, SchemaType: map[string]string{"postgres": "numeric"}},
		{Name: "at", Type: field.TypeTime},
	}
	// BalanceSnapshotsTable holds the schema information for the "balance_snapshots" table.
	BalanceSnapshotsTable = &schema.Table{
		Name:       "balance_snapshots",
		Columns:    BalanceSnapshotsColumns,
		PrimaryKey: []*schema.Column{BalanceSnapshotsColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "balancesnapshot_namespace_at",
				Unique:  false,
				Columns: []*schema.Column{BalanceSnapshotsColumns[1], BalanceSnapshotsColumns[9]},
			},
			{
				Name:    "balancesnapshot_namespace_balance",
				Unique:  false,
				Columns: []*schema.Column{BalanceSnapshotsColumns[1], BalanceSnapshotsColumns[7]},
			},
			{
				Name:    "balancesnapshot_namespace_balance_at",
				Unique:  false,
				Columns: []*schema.Column{BalanceSnapshotsColumns[1], BalanceSnapshotsColumns[7], BalanceSnapshotsColumns[9]},
			},
		},
	}
	// GrantsColumns holds the columns for the "grants" table.
	GrantsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeString, Unique: true, SchemaType: map[string]string{"postgres": "char(26)"}},
		{Name: "namespace", Type: field.TypeString},
		{Name: "metadata", Type: field.TypeJSON, Nullable: true, SchemaType: map[string]string{"postgres": "jsonb"}},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "updated_at", Type: field.TypeTime},
		{Name: "deleted_at", Type: field.TypeTime, Nullable: true},
		{Name: "owner_id", Type: field.TypeString, SchemaType: map[string]string{"postgres": "char(26)"}},
		{Name: "amount", Type: field.TypeFloat64, SchemaType: map[string]string{"postgres": "numeric"}},
		{Name: "priority", Type: field.TypeUint8, Default: 0},
		{Name: "effective_at", Type: field.TypeTime},
		{Name: "expiration", Type: field.TypeJSON, SchemaType: map[string]string{"postgres": "jsonb"}},
		{Name: "expires_at", Type: field.TypeTime},
		{Name: "voided_at", Type: field.TypeTime, Nullable: true},
		{Name: "reset_max_rollover", Type: field.TypeFloat64, SchemaType: map[string]string{"postgres": "numeric"}},
		{Name: "recurrence_max_rollover", Type: field.TypeFloat64, Nullable: true, SchemaType: map[string]string{"postgres": "numeric"}},
		{Name: "recurrence_period", Type: field.TypeEnum, Nullable: true, Enums: []string{"DAILY", "WEEKLY", "MONTHLY", "YEARLY"}},
		{Name: "recurrence_anchor", Type: field.TypeTime, Nullable: true},
	}
	// GrantsTable holds the schema information for the "grants" table.
	GrantsTable = &schema.Table{
		Name:       "grants",
		Columns:    GrantsColumns,
		PrimaryKey: []*schema.Column{GrantsColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "grant_id",
				Unique:  false,
				Columns: []*schema.Column{GrantsColumns[0]},
			},
			{
				Name:    "grant_namespace_owner_id",
				Unique:  false,
				Columns: []*schema.Column{GrantsColumns[1], GrantsColumns[6]},
			},
			{
				Name:    "grant_effective_at_expires_at",
				Unique:  false,
				Columns: []*schema.Column{GrantsColumns[9], GrantsColumns[11]},
			},
		},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		BalanceSnapshotsTable,
		GrantsTable,
	}
)

func init() {
}
