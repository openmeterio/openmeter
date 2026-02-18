{{/*
    This template attaches cursor-based pagination methods to Ent queries.
*/}}
{{ define "cursor" }}

{{ $pkg := base $.Config.Package }}
{{ template "header" $ }}

{{ range $n := $.Nodes }}
    {{ $hasCreatedAt := false }}
    {{ range $f := $n.Fields }}
        {{ if eq $f.Name "created_at" }}
            {{ $hasCreatedAt = true }}
        {{ end }}
    {{ end }}
    {{ if $hasCreatedAt }}
    {{ $receiver := $n.Receiver }}
    // Cursor runs the query and returns a cursor-paginated response.
    // Ordering is always by created_at asc, id asc.
    func ({{ $receiver }} *{{ $n.QueryName }}) Cursor(ctx context.Context, cursor *pagination.Cursor) (pagination.Result[*{{ $n.Name }}], error) {
        if cursor != nil {
            if err := cursor.Validate(); err != nil {
                return pagination.Result[*{{ $n.Name }}]{}, fmt.Errorf("invalid cursor: %w", err)
            }

            {{ $receiver }}.Where(func(s *sql.Selector) {
                s.Where(
                    sql.Or(
                        sql.GT(s.C("created_at"), cursor.Time),
                        sql.And(
                            sql.EQ(s.C("created_at"), cursor.Time),
                            sql.P(func(b *sql.Builder) {
                                b.WriteString("CAST(")
                                b.WriteString(s.C("id"))
                                b.WriteString(" AS TEXT) > ")
                                b.Args(cursor.ID)
                            }),
                        ),
                    ),
                )
            })
        }

        {{ $receiver }}.Order(func(s *sql.Selector) {
            s.OrderBy(sql.Asc(s.C("created_at")), sql.Asc(s.C("id")))
        })

        items, err := {{ $receiver }}.All(ctx)
        if err != nil {
            return pagination.Result[*{{ $n.Name }}]{}, err
        }

        if items == nil {
            items = make([]*{{ $n.Name }}, 0)
        }

        result := pagination.Result[*{{ $n.Name }}]{
            Items: items,
        }

        if len(items) > 0 {
            last := items[len(items)-1]
            result.NextCursor = lo.ToPtr(pagination.NewCursor(last.CreatedAt, fmt.Sprint(last.ID)))
        }

        return result, nil
    }
    {{ end }}
{{ end }}

{{ end }}
