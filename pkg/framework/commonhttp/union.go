package commonhttp

import (
	"encoding/json"
	"fmt"

	"github.com/samber/mo"
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

// wraps mo.Either to be used as a json.Marshaler
type Either[Primary any, Secondary any] struct {
	mo.Either[Primary, Secondary]
}

func (e Either[Primary, Secondary]) MarshalJSON() ([]byte, error) {
	if e.IsLeft() {
		return json.Marshal(e.MustLeft())
	} else if e.IsRight() {
		return json.Marshal(e.MustRight())
	}
	return nil, fmt.Errorf("neither left nor right")
}
