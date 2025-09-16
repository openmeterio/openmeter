package pagination

import "encoding/json"

type Result[T any] struct {
	Page       Page `json:"-"`
	TotalCount int  `json:"totalCount"`
	Items      []T  `json:"items"`
}

// Implement json.Marshaler interface to flatten the Page struct
func (p Result[T]) MarshalJSON() ([]byte, error) {
	type Alias Result[T]
	return json.Marshal(&struct {
		PageSize   int `json:"pageSize"`
		PageNumber int `json:"page"`
		*Alias
	}{
		PageSize:   p.Page.PageSize,
		PageNumber: p.Page.PageNumber,
		Alias:      (*Alias)(&p),
	})
}

// MapResult creates a new Result with the given page, totalCount and items.
func MapResult[Out any, In any](resp Result[In], m func(in In) Out) Result[Out] {
	items := make([]Out, 0, len(resp.Items))
	for _, inItem := range resp.Items {
		items = append(items, m(inItem))
	}

	return Result[Out]{
		Page:       resp.Page,
		TotalCount: resp.TotalCount,
		Items:      items,
	}
}

// MapResultErr is similar to MapResult
// but it allows the mapping function to return an error.
func MapResultErr[Out any, In any](resp Result[In], m func(in In) (Out, error)) (Result[Out], error) {
	items := make([]Out, 0, len(resp.Items))
	for _, inItem := range resp.Items {
		item, err := m(inItem)
		if err != nil {
			return Result[Out]{}, err
		}

		items = append(items, item)
	}

	return Result[Out]{
		Page:       resp.Page,
		TotalCount: resp.TotalCount,
		Items:      items,
	}, nil
}
