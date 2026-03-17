package testutils

import (
	"log/slog"

	"github.com/openmeterio/openmeter/app/common"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
)

type Deps struct {
	AccountService   ledgeraccount.Service
	ResolversService *resolvers.AccountResolver
	HistoricalLedger *historical.Ledger
}

func InitDeps(db *entdb.Client, logger *slog.Logger) (Deps, error) {
	repo := common.NewLedgerAccountRepo(db)
	locker, err := common.NewLocker(logger)
	if err != nil {
		return Deps{}, err
	}

	accountLiveServices := common.NewLedgerAccountLiveServices(locker)
	accountService := common.NewLedgerAccountService(repo, accountLiveServices)
	customerAccountRepo := common.NewLedgerResolversRepo(db)
	accountResolver := common.NewLedgerResolversService(accountService, customerAccountRepo)
	historicalRepo := common.NewLedgerHistoricalRepo(db)
	historicalLedger := common.NewLedgerHistoricalLedger(historicalRepo, accountService, locker)

	return Deps{
		AccountService:   accountService,
		ResolversService: accountResolver,
		HistoricalLedger: historicalLedger,
	}, nil
}
