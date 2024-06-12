package credit

import (
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/creditdriver"
)

type Ledger = credit.Ledger
type Grant = credit.Grant
type HighWatermark = credit.HighWatermark
type Reset = credit.Reset
type Feature = credit.Feature
type Balance = credit.DELETEME_Balance
type LedgerEntryList = credit.LedgerEntryList
type NamespacedLedgerID = credit.NamespacedLedgerID
type NamespacedGrantID = credit.NamespacedGrantID
type NamespacedFeatureID = credit.NamespacedFeatureID

type Connector = credit.Connector
type ListGrantsParams = credit.ListGrantsParams
type ListFeaturesParams = credit.ListFeaturesParams
type ListLedgersParams = credit.ListLedgersParams

type GetLedgerHistoryRequest = creditdriver.GetLedgerHistoryRequest
type GetLedgerBalanceRequest = creditdriver.GetLedgerBalanceRequest
type GetLedgerBalaceHandlerParams = creditdriver.GetLedgerBalaceHandlerParams
type GrantPathParams = creditdriver.GrantPathParams
type ListLedgerGrantsByLedgerParams = creditdriver.ListLedgerGrantsByLedgerParams
