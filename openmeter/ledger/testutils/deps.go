package testutils

import (
	"log/slog"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	accountadapter "github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	accountservice "github.com/openmeterio/openmeter/openmeter/ledger/account/service"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
	historicaladapter "github.com/openmeterio/openmeter/openmeter/ledger/historical/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	resolversadapter "github.com/openmeterio/openmeter/openmeter/ledger/resolvers/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

type Deps struct {
	AccountService   ledgeraccount.Service
	ResolversService *resolvers.AccountResolver
	HistoricalLedger *historical.Ledger
}

func InitDeps(db *entdb.Client, logger *slog.Logger) (Deps, error) {
	repo := accountadapter.NewRepo(db)
	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger,
	})
	if err != nil {
		return Deps{}, err
	}

	historicalRepo := historicaladapter.NewRepo(db)
	accountService := accountservice.New(repo, locker)
	historicalLedger := historical.NewLedger(historicalRepo, accountService, accountService, routingrules.DefaultValidator)
	customerAccountRepo := resolversadapter.NewRepo(db)
	accountResolver := resolvers.NewAccountResolver(resolvers.AccountResolverConfig{
		AccountService: accountService,
		Repo:           customerAccountRepo,
		Locker:         locker,
	})

	return Deps{
		AccountService:   accountService,
		ResolversService: accountResolver,
		HistoricalLedger: historicalLedger,
	}, nil
}
