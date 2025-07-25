package datetime

import (
	"github.com/rickb777/period"
	"github.com/samber/lo"
)

type ISODurationString period.ISOString

func (i ISODurationString) Parse() (ISODuration, error) {
	res, err := period.Parse(string(i))
	if err != nil {
		return ISODuration{}, NewDurationParseError(string(i), err)
	}
	return ISODuration{res}, nil
}

// ParsePtrOrNil parses the ISO8601 string representation of the period or if ISODurationString is nil, returns nil
func (i *ISODurationString) ParsePtrOrNil() (*ISODuration, error) {
	if i == nil {
		return nil, nil
	}

	d, err := i.Parse()
	if err != nil {
		return nil, err
	}

	return lo.ToPtr(d), nil
}

func (i ISODurationString) String() string {
	return string(i)
}
