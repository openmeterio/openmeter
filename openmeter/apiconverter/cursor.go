package apiconverter

import (
	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func ConvertCursor(s apimodels.CursorPaginationCursor) (*pagination.Cursor, error) {
	return pagination.DecodeCursor(s)
}

func ConvertCursorPtr(s *apimodels.CursorPaginationCursor) (*pagination.Cursor, error) {
	if s == nil {
		return nil, nil
	}

	return ConvertCursor(*s)
}
