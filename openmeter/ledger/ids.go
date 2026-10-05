package ledger

import (
	"fmt"
	"slices"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/pkg/models"
)

func WithGroupID(input TransactionGroupInput, id string) TransactionGroupInput {
	return &identifiedGroupInput{TransactionGroupInput: input, id: id, transactions: slices.Clone(input.Transactions())}
}

func WithTransactionID(input TransactionInput, id string) TransactionInput {
	return &identifiedTransactionInput{TransactionInput: input, id: id, entries: slices.Clone(input.EntryInputs())}
}

func WithEntryID(input EntryInput, id string) EntryInput {
	return &identifiedEntryInput{EntryInput: input, id: id}
}

// WithEntryInputs replaces the concrete postings while retaining the transaction's
// ID and metadata. It lets callers assign entry IDs after template resolution.
func WithEntryInputs(input TransactionInput, entries ...EntryInput) TransactionInput {
	return &identifiedTransactionInput{TransactionInput: input, id: input.AssignedID(), entries: slices.Clone(entries)}
}

// PreassignIDs materializes missing transaction and entry IDs without posting or
// mutating the input. Supplied IDs are preserved and validated, so returned entry
// IDs can be referenced before the transaction is committed.
func PreassignIDs(input TransactionInput) (TransactionInput, error) {
	if input == nil {
		return nil, ErrTransactionInputRequired
	}
	if err := ValidateAssignedID(input.AssignedID()); err != nil {
		return nil, fmt.Errorf("transaction ID: %w", err)
	}

	entries := input.EntryInputs()
	identifiedEntries := make([]EntryInput, len(entries))
	for idx, entry := range entries {
		if entry == nil {
			return nil, fmt.Errorf("entries[%d]: entry is required", idx)
		}
		if err := ValidateAssignedID(entry.AssignedID()); err != nil {
			return nil, fmt.Errorf("entries[%d] ID: %w", idx, err)
		}
		identifiedEntries[idx] = entry
		if entry.AssignedID() == "" {
			identifiedEntries[idx] = WithEntryID(entry, ulid.Make().String())
		}
	}

	id := input.AssignedID()
	if id == "" {
		id = ulid.Make().String()
	}
	return &identifiedTransactionInput{TransactionInput: input, id: id, entries: identifiedEntries}, nil
}

type identifiedGroupInput struct {
	TransactionGroupInput
	id           string
	transactions []TransactionInput
}

var _ TransactionGroupInput = (*identifiedGroupInput)(nil)

func (i *identifiedGroupInput) AssignedID() string {
	return i.id
}

func (i *identifiedGroupInput) Transactions() []TransactionInput {
	return slices.Clone(i.transactions)
}

type identifiedTransactionInput struct {
	TransactionInput
	id      string
	entries []EntryInput
}

var _ TransactionInput = (*identifiedTransactionInput)(nil)

func (i *identifiedTransactionInput) AssignedID() string {
	return i.id
}

func (i *identifiedTransactionInput) EntryInputs() []EntryInput {
	return slices.Clone(i.entries)
}

func (i *identifiedTransactionInput) AsGroupInput(namespace string, annotations models.Annotations) TransactionGroupInput {
	// Reuse the underlying group metadata, but retain this wrapper as its
	// transaction so grouping cannot discard assigned IDs or entry overrides.
	group := i.TransactionInput.AsGroupInput(namespace, annotations)
	return &identifiedGroupInput{
		TransactionGroupInput: group,
		id:                    group.AssignedID(),
		transactions:          []TransactionInput{i},
	}
}

type identifiedEntryInput struct {
	EntryInput
	id string
}

var _ EntryInput = (*identifiedEntryInput)(nil)

func (i *identifiedEntryInput) AssignedID() string {
	return i.id
}
