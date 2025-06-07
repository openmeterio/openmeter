package balanceworker

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/models"
)

func resolveSubjectIfExists(ctx context.Context, svc subject.Service, namespacedKey models.NamespacedKey) (subject.Subject, error) {
	subByKey, err := svc.GetByKey(ctx, namespacedKey)
	if err != nil {
		if models.IsGenericNotFoundError(err) {
			return subject.Subject{
				Key: namespacedKey.Key,
			}, nil
		}

		return subject.Subject{}, fmt.Errorf("failed to get subject: %w", err)
	}

	return subByKey, nil
}
