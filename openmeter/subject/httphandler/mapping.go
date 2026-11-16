package httpdriver

import (
	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/subject/apiconvert"
)

func FromSubject(s subject.Subject) apiv1models.Subject {
	return apiconvert.ToAPISubject(s)
}
