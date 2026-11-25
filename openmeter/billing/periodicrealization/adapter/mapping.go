package adapter

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func fromDBBillingPeriodicRealizationRun(row *entdb.BillingPeriodicRealizationRun) periodicrealization.Run {
	return periodicrealization.Run{
		NamespacedID: models.NamespacedID{
			Namespace: row.Namespace,
			ID:        row.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: row.CreatedAt.In(time.UTC),
			UpdatedAt: row.UpdatedAt.In(time.UTC),
			DeletedAt: convert.TimePtrIn(row.DeletedAt, time.UTC),
		},
		PreviousRunID: row.PreviousRunID,
		PeriodTo:      row.PeriodTo.In(time.UTC),
		RealizeAfter:  row.RealizeAfter.In(time.UTC),
		Status:        row.Status,
	}
}
