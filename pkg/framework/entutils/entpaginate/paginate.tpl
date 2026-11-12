{{/*
    This template attaches simple type safe options to run window functions on a query.
*/}}
{{ define "paginate" }}


{{/* Add the base header for the generated file */}}
{{ $pkg := base $.Config.Package }}
{{ template "header" $ }}

{{/* Loop over all nodes and implement "pagination.Paginator" for "XQuery" and "XSelect" */}}
{{ range $n := $.Nodes }}
    {{ $receiver := $n.Receiver }}
    // Paginate runs the query and returns a paginated response.
    // If page is its zero value, it returns all items and derives the total from them.
    func ({{ $receiver }} *{{ $n.QueryName }}) Paginate(ctx context.Context, page pagination.Page) (pagination.Result[*{{ $n.Name }}], error) {
        // Get the limit and offset
        limit, offset := page.Limit(), page.Offset()

        // Unset previous pagination settings
        zero := 0
        {{ $receiver }}.ctx.Offset = &zero
        {{ $receiver }}.ctx.Limit = &zero

        pagedResponse := pagination.Result[*{{ $n.Name }}]{
            Page: page,
        }

        if page.IsZero() {
            // All matching rows are loaded, so a separate count would repeat the work.
            items, err := {{ $receiver }}.All(ctx)
            if err != nil {
                return pagedResponse, err
            }

            pagedResponse.Items = items
            pagedResponse.TotalCount = len(items)
            return pagedResponse, nil
        }

        // Create duplicate of the query to run for
        countQuery := {{ $receiver }}.Clone()
        pagedQuery := {{ $receiver }}

        // Unset select for count query
        countQuery.ctx.Fields = []string{}

        // Unset ordering for count query
        countQuery.order = nil

        // Get the total count
        count, err := countQuery.Count(ctx)
        if err != nil {
            return pagedResponse, fmt.Errorf("failed to get count: %w", err)
        }
        pagedResponse.TotalCount = count

        // If there are no items, return the empty response early
        if count == 0 {
            // Items should be [] not null.
            pagedResponse.Items = make([]*{{ $n.Name }}, 0)
            return pagedResponse, nil
        }

        // Set the limit and offset
        pagedQuery.ctx.Limit = &limit
        pagedQuery.ctx.Offset = &offset


        // Get the paged items
        items, err := pagedQuery.All(ctx)
        pagedResponse.Items = items
        return pagedResponse, err
    }

    // type check
    var _ pagination.Paginator[*{{ $n.Name }}] = (*{{ $n.QueryName }})(nil)
{{ end }}

{{ end }}
