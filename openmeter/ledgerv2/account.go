package ledgerv2

import (
	"github.com/openmeterio/openmeter/pkg/models"
)

// AccountData is a simple data transfer object for the Account entity.
type AccountData struct {
	ID          models.NamespacedID
	Annotations models.Annotations
	models.ManagedModel
	AccountType AccountType
}

type OrganizationalAccount struct {
	AccountData
}

type CustomerAccount struct {
	AccountData

	CustomerID string
}

type Account struct {
	t AccountType
	o *OrganizationalAccount
	c *CustomerAccount
}

func (a *Account) AsOrganizationalAccount() *OrganizationalAccount {
	return a.o
}

func (a *Account) AsCustomerAccount() *CustomerAccount {
	return a.c
}
