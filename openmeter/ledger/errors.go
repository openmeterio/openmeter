package ledger

import "github.com/openmeterio/openmeter/pkg/models"

// TODO: Better names and codes

const ErrCodeInvalidTransactionTotal models.ErrorCode = "invalid_transaction_total"

var ErrInvalidTransactionTotal = models.NewValidationIssue(
	ErrCodeInvalidTransactionTotal,
	"transaction total is invalid, credits and debits must sum to 0",
)

var ErrCodeLedgerQueryInvalid models.ErrorCode = "ledger_query_invalid"

var ErrLedgerQueryInvalid = models.NewValidationIssue(
	ErrCodeLedgerQueryInvalid,
	"ledger query is invalid",
)

const ErrCodeCurrencyInvalid models.ErrorCode = "ledger_currency_invalid"

var ErrCurrencyInvalid = models.NewValidationIssue(
	ErrCodeCurrencyInvalid,
	"ledger currency is invalid",
)

const ErrCodeCreditPriorityInvalid models.ErrorCode = "ledger_credit_priority_invalid"

var ErrCreditPriorityInvalid = models.NewValidationIssue(
	ErrCodeCreditPriorityInvalid,
	"ledger credit priority is invalid",
)

const ErrCodeCostBasisInvalid models.ErrorCode = "ledger_cost_basis_invalid"

var ErrCostBasisInvalid = models.NewValidationIssue(
	ErrCodeCostBasisInvalid,
	"ledger cost basis is invalid",
)

const ErrCodeRoutingKeyVersionInvalid models.ErrorCode = "ledger_routing_key_version_invalid"

var ErrRoutingKeyVersionInvalid = models.NewValidationIssue(
	ErrCodeRoutingKeyVersionInvalid,
	"ledger routing key version is invalid",
)

const ErrCodeRoutingKeyVersionUnsupported models.ErrorCode = "ledger_routing_key_version_unsupported"

var ErrRoutingKeyVersionUnsupported = models.NewValidationIssue(
	ErrCodeRoutingKeyVersionUnsupported,
	"ledger routing key version is unsupported",
)

const ErrCodeResolutionScopeInvalid models.ErrorCode = "ledger_resolution_scope_invalid"

var ErrResolutionScopeInvalid = models.NewValidationIssue(
	ErrCodeResolutionScopeInvalid,
	"ledger resolution scope is invalid",
)

const ErrCodeResolutionTemplateUnknown models.ErrorCode = "ledger_resolution_template_unknown"

var ErrResolutionTemplateUnknown = models.NewValidationIssue(
	ErrCodeResolutionTemplateUnknown,
	"ledger transaction template type is unknown",
)

const ErrCodeCustomerAccountMissing models.ErrorCode = "ledger_customer_account_missing"

var ErrCustomerAccountMissing = models.NewValidationIssue(
	ErrCodeCustomerAccountMissing,
	"required customer ledger account is missing",
)

const ErrCodeTransactionInputRequired models.ErrorCode = "ledger_transaction_input_required"

var ErrTransactionInputRequired = models.NewValidationIssue(
	ErrCodeTransactionInputRequired,
	"transaction input is required",
)

const ErrCodeEntryInvalid models.ErrorCode = "ledger_entry_invalid"

var ErrEntryInvalid = models.NewValidationIssue(
	ErrCodeEntryInvalid,
	"ledger entry is invalid",
)

const ErrCodeAddressInvalid models.ErrorCode = "ledger_address_invalid"

var ErrAddressInvalid = models.NewValidationIssue(
	ErrCodeAddressInvalid,
	"ledger posting address is invalid",
)

const ErrCodeListTransactionsInputInvalid models.ErrorCode = "ledger_list_transactions_input_invalid"

var ErrListTransactionsInputInvalid = models.NewValidationIssue(
	ErrCodeListTransactionsInputInvalid,
	"ledger list transactions input is invalid",
)

const ErrCodeTransactionGroupEmpty models.ErrorCode = "ledger_transaction_group_empty"

var ErrTransactionGroupEmpty = models.NewValidationIssue(
	ErrCodeTransactionGroupEmpty,
	"ledger transaction group must contain at least one transaction",
)

const ErrCodeRoutingRuleViolated models.ErrorCode = "ledger_routing_rule_violated"

var ErrRoutingRuleViolated = models.NewValidationIssue(
	ErrCodeRoutingRuleViolated,
	"ledger routing rule violated",
)
