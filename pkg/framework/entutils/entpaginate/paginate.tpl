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
    // If page is its 0 value then it will return all the items and populate the response page accordingly.
    func ({{ $receiver }} *{{ $n.QueryName }}) Paginate(ctx context.Context, page pagination.Page) (pagination.Result[*{{ $n.Name }}], error) {
        // Get the limit and offset
        limit, offset := page.Limit(), page.Offset()

        // Unset previous pagination settings
        zero := 0
        {{ $receiver }}.ctx.Offset = &zero
        {{ $receiver }}.ctx.Limit = &zero

        // Create duplicate of the query to run for
        countQuery := {{ $receiver }}.Clone()
        pagedQuery := {{ $receiver }}


        // Unset ordering for count query
        countQuery.order = nil

        pagedResponse := pagination.Result[*{{ $n.Name }}]{
            Page: page,
        }

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

        // If page is its 0 value then return all the items
        if page.IsZero() {
            offset = 0
            limit = count
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
