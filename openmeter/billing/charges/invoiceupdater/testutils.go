package invoiceupdater

import (
	"context"
	"errors"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

type unimplementedUpdater struct {
	t testing.TB
}

func NewUnimplementedUpdater(t testing.TB) Updater {
	return unimplementedUpdater{
		t: t,
	}
}

func (u unimplementedUpdater) ApplyPatches(context.Context, customer.CustomerID, Patches) error {
	if u.t != nil {
		u.t.Helper()
	}

	return errors.New("invoice updater is not implemented")
}
