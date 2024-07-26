package commonhttp

import (
	"encoding/json"
)

type Union[Primary any, Secondary any] struct {
	Option1 *Primary
	Option2 *Secondary
}

// Implements json.Marshaler with primary having precedence.
func (u Union[Primary, Secondary]) MarshalJSON() ([]byte, error) {
	if u.Option1 != nil {
		return json.Marshal(u.Option1)
	}
	if u.Option2 != nil {
		return json.Marshal(u.Option2)
	}
	// if nothing is set we return empty
	return []byte{}, nil
}
