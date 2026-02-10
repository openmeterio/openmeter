package ledger

import "github.com/openmeterio/openmeter/pkg/models"

// TODO: Better names and codes

const ErrCodeInvalidTransactionTotal models.ErrorCode = "invalid_transaction_total"

var ErrInvalidTransactionTotal = models.NewValidationIssue(
	ErrCodeInvalidTransactionTotal,
	"transaction total is invalid, credits and debits must sum to 0",
)

const ErrCodeCreditAccountBalanceIsNegative models.ErrorCode = "credit_account_balance_is_negative"

var ErrCreditAccountBalanceIsNegative = models.NewValidationIssue(
	ErrCodeCreditAccountBalanceIsNegative,
	"credit account balance is negative",
)
