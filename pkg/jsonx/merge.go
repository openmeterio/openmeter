package jsonx

import "github.com/valyala/fastjson"

func Merge(base, overrides []byte) ([]byte, error) {
	baseParsed, err := fastjson.ParseBytes(base)
	if err != nil {
		return nil, err
	}

	baseObject, err := baseParsed.Object()
	if err != nil {
		return nil, err
	}

	overridesParsed, err := fastjson.ParseBytes(overrides)
	if err != nil {
		return nil, err
	}

	overridesObject, err := overridesParsed.Object()
	if err != nil {
		return nil, err
	}

	overridesObject.Visit(func(key []byte, v *fastjson.Value) {
		baseObject.Set(string(key), v)
	})

	return baseObject.MarshalTo(make([]byte, 0, len(base)+len(overrides))), nil
}
