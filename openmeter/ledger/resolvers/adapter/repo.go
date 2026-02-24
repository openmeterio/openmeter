package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgercustomeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgercustomeraccount"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
)

type repo struct {
	db *entdb.Client
}

var _ resolvers.Repo = (*repo)(nil)

func NewRepo(db *entdb.Client) resolvers.Repo {
	return &repo{db: db}
}

func (r *repo) CreateCustomerAccount(ctx context.Context, input resolvers.CreateCustomerAccountInput) error {
	_, err := r.db.LedgerCustomerAccount.Create().
		SetNamespace(input.CustomerID.Namespace).
		SetCustomerID(input.CustomerID.ID).
		SetAccountType(input.AccountType).
		SetAccountID(input.AccountID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create ledger customer account: %w", err)
	}

	return nil
}

func (r *repo) GetCustomerAccountIDs(ctx context.Context, customerID customer.CustomerID) (map[ledger.AccountType]string, error) {
	entities, err := r.db.LedgerCustomerAccount.Query().
		Where(
			ledgercustomeraccountdb.Namespace(customerID.Namespace),
			ledgercustomeraccountdb.CustomerID(customerID.ID),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger customer accounts: %w", err)
	}

	result := make(map[ledger.AccountType]string, len(entities))
	for _, entity := range entities {
		result[entity.AccountType] = entity.AccountID
	}

	return result, nil
}
