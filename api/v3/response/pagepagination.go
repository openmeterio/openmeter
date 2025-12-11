package response

type PagePaginationResponse[T any] struct {
	Data []T      `json:"data"`
	Meta PageMeta `json:"meta"`
}

type PageMeta struct {
	Page PageMetaPage `json:"page"`
}

type PageMetaPage struct {
	Size           int  `json:"size"`
	Number         int  `json:"number"`
	Total          *int `json:"total,omitempty"`
	EstimatedTotal *int `json:"estimatedTotal,omitempty"`
}

func NewPagePaginationResponse[T any](items []T, page PageMetaPage) PagePaginationResponse[T] {
	return PagePaginationResponse[T]{
		Data: items,
		Meta: PageMeta{
			Page: page,
		},
	}
}
