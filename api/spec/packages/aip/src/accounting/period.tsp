import "../shared/index.tsp";

namespace Accounting;

/**
 * Lifecycle status of an accounting period.
 *
 * - `open`: the current posting bucket. New transactions can be posted to this
 *   period. Exactly one period per namespace is `open` at any time.
 * - `closing`: membership is frozen — no new transactions can post to this period
 *   — but balance snapshots and the seal have not yet been committed. At most one
 *   period per namespace is `closing` at any time.
 * - `closed`: the period is sealed. `closed_at`, `sealed_at`, and (when sealing is
 *   enabled) `seal_hash` are set. Transaction membership and the closing balance
 *   snapshots are immutable.
 */
@friendlyName("AccountingPeriodStatus")
@summary("Accounting period status")
enum AccountingPeriodStatus {
  // The current posting bucket for new transactions.
  Open: "open",

  // Membership is frozen but the seal has not yet been committed.
  Closing: "closing",

  // The period is sealed and immutable.
  Closed: "closed",
}

/**
 * An accounting period is a sequential, non-overlapping window of ledger activity
 * within a namespace. Closing a period freezes its transaction membership and
 * (eventually) seals an authoritative closing balance.
 *
 * Late-arriving transactions whose `booked_at` falls inside an already-closed
 * period are posted to the current open period with an `adjusts_period_id`
 * backpointer; they do not mutate the closed period.
 */
@friendlyName("AccountingPeriod")
@summary("Accounting period")
model AccountingPeriod {
  ...Shared.ResourceImmutable;

  /**
   * Start of the period (inclusive).
   *
   * For the first period in a namespace this is the namespace creation time. For
   * every subsequent period, equals the previous period's `end_at`. Immutable once
   * set.
   */
  @visibility(Lifecycle.Read)
  @summary("Start of period")
  start_at: Shared.DateTime;

  /**
   * End of the period (exclusive).
   *
   * Unset while the period is `open`. Fixed at the moment a close is initiated, and
   * immutable thereafter.
   */
  @visibility(Lifecycle.Read)
  @summary("End of period")
  end_at?: Shared.DateTime;

  /**
   * Current lifecycle status of the period.
   */
  @visibility(Lifecycle.Read)
  @summary("Status")
  status: AccountingPeriodStatus;

  /**
   * Timestamp when the close was initiated (open → closing transition). Set on
   * `closing` and `closed` periods.
   */
  @visibility(Lifecycle.Read)
  @summary("Closing started at")
  closing_started_at?: Shared.DateTime;

  /**
   * Timestamp when the period was marked `closed` (closing → closed transition). Set
   * only on `closed` periods.
   */
  @visibility(Lifecycle.Read)
  @summary("Closed at")
  closed_at?: Shared.DateTime;

  /**
   * Timestamp when the seal hash was committed. Set only on `closed` periods that
   * have a seal hash.
   */
  @visibility(Lifecycle.Read)
  @summary("Sealed at")
  sealed_at?: Shared.DateTime;

  /**
   * Cryptographic hash sealing the period. The hash covers the period's frozen
   * transaction membership and closing balance snapshots in a deterministic,
   * versioned input format described by `seal_algorithm` and `seal_input_version`.
   *
   * Set only on `closed` periods that have been sealed.
   */
  @visibility(Lifecycle.Read)
  @summary("Seal hash")
  seal_hash?: string;

  /**
   * Algorithm used to compute `seal_hash`. Example: `"sha256"`.
   */
  @visibility(Lifecycle.Read)
  @summary("Seal algorithm")
  seal_algorithm?: string;

  /**
   * Version of the seal input format. Versioning the input format lets verifiers
   * reproduce a historical seal even after the format evolves.
   */
  @visibility(Lifecycle.Read)
  @summary("Seal input version")
  seal_input_version?: string;
}
