package ledger

type RoutingValidator interface {
	ValidateEntries(entries []EntryInput) error
}
