package ref

import "fmt"

type IDOrKey struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

func (i IDOrKey) Validate() error {
	if i.ID == "" && i.Key == "" {
		return fmt.Errorf("either id or key is required")
	}

	return nil
}
