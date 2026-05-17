# Example on using the entselectedparse extension

```go
func (r *repo) ListEntries(ctx context.Context, input ledgerhistorical.ListEntriesInput) (pagination.Result[ledgerhistorical.EntryData], error) {
	query := r.db.LedgerEntry.Query()

	if input.Filters.Account != nil {
		address := input.Filters.Account
		query = query.Where(
			ledgerentrydb.AccountID(address.ID().ID),
			ledgerentrydb.Namespace(address.ID().Namespace),
		)
	}

	if input.Filters.TransactionID != nil {
		query = query.Where(ledgerentrydb.TransactionID(*input.Filters.TransactionID))
	}

	if input.Filters.BookedAtPeriod != nil {
		transactionPredicates := make([]predicate.LedgerTransaction, 0, 2)
		if input.Filters.BookedAtPeriod.From != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtGTE(*input.Filters.BookedAtPeriod.From))
		}
		if input.Filters.BookedAtPeriod.To != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtLT(*input.Filters.BookedAtPeriod.To))
		}
		if len(transactionPredicates) > 0 {
			query = query.Where(ledgerentrydb.HasTransactionWith(transactionPredicates...))
		}
	}

	if input.Cursor != nil {
		query = query.Where(
			ledgerentrydb.Or(
				ledgerentrydb.CreatedAtGT(input.Cursor.Time),
				ledgerentrydb.And(
					ledgerentrydb.CreatedAt(input.Cursor.Time),
					ledgerentrydb.IDGT(input.Cursor.ID),
				),
			),
		)
	}

	if input.Expand.Dimensions {
		query = query.Unique(false)
		query = query.Where(func(s *sql.Selector) {
			baseWhere := s.P()
			e := sql.Table(ledgerentrydb.Table)
			paged := sql.Select(e.Columns(ledgerentrydb.Columns...)...).From(e)
			if baseWhere != nil {
				paged.Where(baseWhere)
			}
			paged.OrderBy(
				sql.Asc(e.C(ledgerentrydb.FieldCreatedAt)),
				sql.Asc(e.C(ledgerentrydb.FieldID)),
			)
			if input.Limit > 0 {
				paged.Limit(input.Limit)
			}
			paged.As("e_paged")

			s.SetP(nil)
			s.SetDistinct(false)
			s.From(paged)

			ePaged := sql.Table("e_paged")
			ePaged.SetDialect(s.Dialect())

			selectColumns := make([]string, 0, len(ledgerentrydb.Columns))
			for _, col := range ledgerentrydb.Columns {
				selectColumns = append(selectColumns, ePaged.C(col))
			}
			s.Select(selectColumns...)

			d := sql.Table(ledgerdimensiondb.Table).As("d")

			s.LeftJoin(d).OnP(sql.And(
				sql.ColumnsEQ(d.C(ledgerdimensiondb.FieldNamespace), ePaged.C(ledgerentrydb.FieldNamespace)),
				sql.ExprP(fmt.Sprintf("%s = ANY(%s)", d.C(ledgerdimensiondb.FieldID), ePaged.C(ledgerentrydb.FieldDimensionIds))),
			))

			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldID), "dimension_id")
			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldNamespace), "dimension_namespace")
			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldAnnotations), "dimension_annotations")
			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldCreatedAt), "dimension_created_at")
			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldUpdatedAt), "dimension_updated_at")
			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldDeletedAt), "dimension_deleted_at")
			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldDimensionKey), "dimension_dimension_key")
			s.AppendSelectAs(d.C(ledgerdimensiondb.FieldDimensionValue), "dimension_dimension_value")
			s.OrderBy(
				sql.Asc(ePaged.C(ledgerentrydb.FieldCreatedAt)),
				sql.Asc(ePaged.C(ledgerentrydb.FieldID)),
				sql.Asc(d.C(ledgerdimensiondb.FieldDimensionKey)),
			)
		})
	} else {
		query = query.Order(
			ledgerentrydb.ByCreatedAt(sql.OrderAsc()),
			ledgerentrydb.ByID(sql.OrderAsc()),
		)

		if input.Limit > 0 {
			query = query.Limit(input.Limit)
		}
	}

	rows, err := query.All(ctx)
	if err != nil {
		return pagination.Result[ledgerhistorical.EntryData]{}, fmt.Errorf("failed to list ledger entries: %w", err)
	}

	if !input.Expand.Dimensions {
		items := lo.Map(rows, func(entity *db.LedgerEntry, _ int) ledgerhistorical.EntryData {
			dimensionIDs := fromTextArray(entity.DimensionIds)

			return ledgerhistorical.EntryData{
				ID:            entity.ID,
				Namespace:     entity.Namespace,
				Annotations:   entity.Annotations,
				CreatedAt:     entity.CreatedAt,
				AccountID:     entity.AccountID,
				AccountType:   entity.AccountType,
				DimensionIDs:  dimensionIDs,
				Amount:        entity.Amount,
				TransactionID: entity.TransactionID,
			}
		})

		var nextCursor *pagination.Cursor
		if len(items) > 0 {
			last := items[len(items)-1]
			nextCursor = lo.ToPtr(pagination.NewCursor(last.CreatedAt, last.ID))
		}

		return pagination.Result[ledgerhistorical.EntryData]{
			Items:      items,
			NextCursor: nextCursor,
		}, nil
	}

	itemsByID := make(map[string]*ledgerhistorical.EntryData)
	order := make([]string, 0, len(rows))

	for _, row := range rows {
		item, exists := itemsByID[row.ID]
		if !exists {
			dimensionIDs := fromTextArray(row.DimensionIds)

			item = &ledgerhistorical.EntryData{
				ID:                 row.ID,
				Namespace:          row.Namespace,
				Annotations:        row.Annotations,
				CreatedAt:          row.CreatedAt,
				AccountID:          row.AccountID,
				AccountType:        row.AccountType,
				DimensionIDs:       dimensionIDs,
				DimensionsExpanded: map[string]*ledgeraccount.Dimension{},
				Amount:             row.Amount,
				TransactionID:      row.TransactionID,
			}
			itemsByID[row.ID] = item
			order = append(order, row.ID)
		}

		dimension, err := db.ParseLedgerDimensionFromSelectedValues("dimension_", row.Value)
		if err != nil {
			return pagination.Result[ledgerhistorical.EntryData]{}, fmt.Errorf("failed to parse joined dimension: %w", err)
		}
		if dimension == nil {
			continue
		}

		item.DimensionsExpanded[dimension.DimensionKey] = &ledgeraccount.Dimension{
			ID: models.NamespacedID{
				Namespace: dimension.Namespace,
				ID:        dimension.ID,
			},
			Annotations: dimension.Annotations,
			ManagedModel: models.ManagedModel{
				CreatedAt: dimension.CreatedAt,
				UpdatedAt: dimension.UpdatedAt,
				DeletedAt: dimension.DeletedAt,
			},
			DimensionKey:   dimension.DimensionKey,
			DimensionValue: dimension.DimensionValue,
		}
	}

	items := make([]ledgerhistorical.EntryData, 0, len(order))
	for _, id := range order {
		items = append(items, *itemsByID[id])
	}

	var nextCursor *pagination.Cursor
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = lo.ToPtr(pagination.NewCursor(last.CreatedAt, last.ID))
	}

	return pagination.Result[ledgerhistorical.EntryData]{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}
```
