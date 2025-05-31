package adapter

import (
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/subject"
)

func New(ent *db.Client) subject.Service {
	return &adapter{
		ent: ent,
	}
}

var _ subject.Service = (*adapter)(nil)

type adapter struct {
	ent *db.Client
}
