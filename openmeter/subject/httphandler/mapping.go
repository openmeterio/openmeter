package httpdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/subject/apiconvert"
)

func FromSubject(s subject.Subject) api.Subject {
	return apiconvert.ToAPISubject(s)
}
