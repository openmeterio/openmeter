# ledger

<!-- archie:ai-start -->

> Double-entry ledger for customer financial balances (FBO, Receivable, Accrued) and shared business accounts (Wash, Earnings, Brokerage, Breakage). Transaction inputs are constructed exclusively via transactions.ResolveTransactions with typed templates (enforcing debit=credit), and noop/ supplies compile-checked zero-value implementations wired when credits.enabled=false.

## Patterns

**TransactionInput constructed only via transaction templates** — transactions/ is the ONLY layer that builds EntryInput/TransactionInput. Callers pass named value-struct templates (CustomerTransactionTemplate, OrgTransactionTemplate) to ResolveTransactions, which resolves abstract intents into postings and routes sub-accounts; corrections reverse via the annotation-stored template name. (`transactions/resolve.go: ResolveTransactions iterates templates, calls resolve(), then ledger.CommitGroup`)
**BookTransaction always inside transaction.Run with routing validation** — historical/ledger.go (the engine implementing ledger.Ledger + BalanceQuerier) pre-locks all accounts, calls ValidateTransactionInputWith(ctx, tx, routingValidator) using routingrules/, then repo.BookTransaction — all inside transaction.Run. Bypassing this loses the account lock and routing invariants. (`historical/ledger.go: transaction.Run(..., func(...) { ValidateTransactionInputWith; repo.BookTransaction })`)
**Canonical sub-account routing key from Route** — Sub-account keys are derived from a Route value via ledger.BuildRoutingKey/BuildRoutingKeyV1 (normalizes decimals and nil fields). Never hand-build routing key strings or AddressData.RoutingKey — it breaks canonical uniqueness. routingrules/ validates permitted account-type pairings before persistence. (`routing.go: BuildRoutingKeyV1(Route{Currency:'USD', CostBasis:..., ...})`)
**credits.enabled=false routes everything to noop/ with compile-time assertions** — app/common wires ledger.Ledger, ledger.AccountResolver, ledgeraccount.Service, and the namespace/customer hooks to noop/ value-structs when credits are disabled. Every new ledger interface method needs a zero-value noop, proven by var _ Interface = Type{} assertions in noop/noop.go, so billing/customer code is structurally identical regardless of the flag. (`noop/noop.go: var _ ledger.Ledger = Ledger{}; var _ ledger.AccountResolver = AccountResolver{}`)
**Charge lifecycle bridged in via chargeadapter, not by charge packages importing ledger** — chargeadapter/ implements per-charge-type Handler interfaces (defined in the charge sub-packages) translating charge state transitions into templates committed via CommitGroup, annotating every call with ChargeAnnotations and routing FBO collection through collector.Service. This keeps charge packages free of ledger imports. (`chargeadapter/helpers.go sets ChargeAnnotations on every CommitGroup input`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/ledger/account.go` | AccountType constants, CustomerAccounts/BusinessAccounts structs, AccountResolver interface. | GetCustomerAccounts assumes provisioned accounts — call EnsureCustomerAccounts first (resolvers/ returns ErrCustomerAccountMissing otherwise). |
| `openmeter/ledger/routing.go` | Route struct, BuildRoutingKeyV1, RoutingKey — the canonical hash key for sub-account routing. | CostBasis decimal is canonicalized by trimming trailing zeros; construct Route via typed field assignment, never string parsing. |
| `openmeter/ledger/validations.go` | ValidateInvariance (sum-to-zero debit/credit check), ValidateEntryInput, ValidateTransactionInputWith — run before every BookTransaction. | Use alpacadecimal, not float64; ValidateInvariance requires all entry amounts to sum to zero. |
| `openmeter/ledger/noop/noop.go` | Zero-value noop implementations of all ledger interfaces for credits.enabled=false. | Missing a method breaks compile when credits are disabled; never return nil for interface-typed fields inside CustomerAccounts/BusinessAccounts (callers dereference without nil checks). |

## Anti-Patterns

- Constructing EntryInput/TransactionInput outside the transactions/ package — bypasses sub-account routing and routing-rule validation.
- Registering CustomerLedgerHook or the namespace ledger handler when credits.enabled=false — these are real DB write paths; the ledgernoop.* variants must be used.
- Calling repo.BookTransaction (or ledger.CommitGroup for FBO collection) outside transaction.Run / outside collector.Service — loses lock, validation atomicity, and lineage correctness.
- Adding a transaction template without registering it in transactionTemplateByName — it becomes uncorrectable at correction time.
- Using context.Background() anywhere in the ledger — drops the Ent transaction and OTel spans.

## Decisions

- **transactions/ is the only permitted construction site for EntryInput/TransactionInput.** — Centralizing construction enforces routing-rule compliance and makes corrections (reversals) traceable via annotation-stored template names.
- **noop/ provides compile-time-checked no-ops for all ledger interfaces.** — Lets billing and customer domains stay structurally identical regardless of credits.enabled — no nil checks scattered through business logic.
- **Tri-tuple cursor (bookedAt, createdAt, ID) for transaction pagination instead of an auto-increment offset.** — Ledger transactions are append-only but may be inserted out of clock order; a composite cursor with ID tie-breaking gives stable pagination.

<!-- archie:ai-end -->
