package charges

// CreditNotesSupportedByLineUpdater controls whether charge-backed immutable
// invoice-line proration can materialize replacement gathering lines. The
// default behavior is false until the invoice line updater supports credit
// notes for correcting immutable invoice history.
var CreditNotesSupportedByLineUpdater = false
