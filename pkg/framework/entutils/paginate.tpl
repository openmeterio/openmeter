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
    func ({{ $receiver }} *{{ $n.QueryName }}) Paginate(ctx context.Context, page pagination.Page) (pagination.PagedResponse[*{{ $n.Name }}], error) {
        // Get the limit and offset
        limit, offset := page.Limit(), page.Offset()

        // Unset previous pagination settings
        zero := 0
        {{ $receiver }}.ctx.Offset = &zero
        {{ $receiver }}.ctx.Limit = &zero

        // Create duplicate of the query to run for
        countQuery := {{ $receiver }}.Clone()
        pagedQuery := {{ $receiver }}

        // Set the limit and offset
        pagedQuery.ctx.Limit = &limit
        pagedQuery.ctx.Offset = &offset

        // Unset ordering for count query
        countQuery.order = nil

        pagedResponse := pagination.PagedResponse[*{{ $n.Name }}]{
            Page: page,
        }

        // Get the total count
        count, err := countQuery.Count(ctx)
        if err != nil {
            return pagedResponse, fmt.Errorf("failed to get count: %w", err)
        }
        pagedResponse.TotalCount = count

        // Get the paged items
        items, err := pagedQuery.All(ctx)
        pagedResponse.Items = items
        return pagedResponse, err
    }

    // type check
    var _ pagination.Paginator[*{{ $n.Name }}] = (*{{ $n.QueryName }})(nil)
{{ end }}

{{ end }}
