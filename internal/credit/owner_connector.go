package credit

import "github.com/openmeterio/openmeter/internal/streaming"

type OwnerConnector interface {
	GetOwnerQueryParams(owner NamespacedGrantOwner) (namespace string, defaultParams streaming.QueryParams, err error)
}
