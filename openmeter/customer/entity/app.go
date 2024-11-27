package customerentity

import (
	"context"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

type App interface {
	// ValidateCustomer validates if the app can run for the given customer
	ValidateCustomer(ctx context.Context, customer *Customer, capabilities []appentitybase.CapabilityType) error
}

// // GetApp returns the app from the app entity
// func GetApp(app appentity.App) (App, error) {
// 	customerApp, ok := app.(App)
// 	if !ok {
// 		return nil, CustomerAppError{
// 			AppID:   app.GetID(),
// 			AppType: app.GetType(),
// 			Err:     fmt.Errorf("is not a customer app"),
// 		}
// 	}

// 	return customerApp, nil
// }
