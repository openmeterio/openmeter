package ledger

type AccountType string

// Customer Accounts
const (
	AccountTypeCustomerFBO        AccountType = "customer_fbo" // is this the right name?
	AccountTypeCustomerReceivable AccountType = "customer_receivable"
	AccountTypeCustomerBreakage   AccountType = "customer_breakage"
)

// Shared Business Accounts
const (
	AccountTypeWash      AccountType = "wash"
	AccountTypeEarnings  AccountType = "earnings"
	AccountTypeBrokerage AccountType = "brokerage"
)
