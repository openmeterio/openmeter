package historical

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

const transactionGroupFingerprintVersion = "v1"

var transactionFingerprintAnnotationKeys = []string{
	ledger.AnnotationChargeNamespace,
	ledger.AnnotationChargeID,
	ledger.AnnotationSubscriptionID,
	ledger.AnnotationSubscriptionPhaseID,
	ledger.AnnotationSubscriptionItemID,
	ledger.AnnotationFeatureID,
	ledger.AnnotationCollectionType,
	ledger.AnnotationCollectionSourceOrder,
	ledger.AnnotationBreakageKind,
	ledger.AnnotationBreakageRecordID,
	ledger.AnnotationBreakagePlanID,
}

type transactionGroupFingerprintPayload struct {
	Transactions []transactionFingerprint `json:"transactions"`
}

type transactionFingerprint struct {
	BookedAt    string                  `json:"bookedAt"`
	Template    string                  `json:"template,omitempty"`
	Direction   string                  `json:"direction,omitempty"`
	Annotations []fingerprintAnnotation `json:"annotations,omitempty"`
	Entries     []entryFingerprint      `json:"entries"`
}

type fingerprintAnnotation struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type entryFingerprint struct {
	AccountType    ledger.AccountType        `json:"accountType"`
	SubAccountID   string                    `json:"subAccountId"`
	RouteKey       string                    `json:"routeKey"`
	RouteVersion   ledger.RoutingKeyVersion  `json:"routeVersion"`
	Amount         string                    `json:"amount"`
	IdentityKey    string                    `json:"identityKey"`
	SchemaVersion  ledger.EntrySchemaVersion `json:"schemaVersion"`
	SourceChargeID *string                   `json:"sourceChargeId,omitempty"`
	SpendChargeID  *string                   `json:"spendChargeId,omitempty"`
	sortKey        string
}

func validateTransactionGroupIdempotencyKey(key *string) error {
	if key == nil {
		return nil
	}

	if *key == "" || utf8.RuneCountInString(*key) > ledger.TransactionGroupIdempotencyKeyMaxLength {
		return ledger.ErrTransactionGroupIdempotencyKeyInvalid.WithAttrs(models.Attributes{
			"max_length": ledger.TransactionGroupIdempotencyKeyMaxLength,
		})
	}

	return nil
}

func transactionGroupFingerprint(group ledger.TransactionGroupInput) (string, error) {
	payload := transactionGroupFingerprintPayload{
		Transactions: make([]transactionFingerprint, 0, len(group.Transactions())),
	}

	for transactionIndex, transactionInput := range group.Transactions() {
		transaction, err := fingerprintTransaction(transactionInput)
		if err != nil {
			return "", fmt.Errorf("transaction %d: %w", transactionIndex, err)
		}

		payload.Transactions = append(payload.Transactions, transaction)
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("serialize fingerprint input: %w", err)
	}

	digest := sha256.Sum256(serialized)

	return transactionGroupFingerprintVersion + ":" + hex.EncodeToString(digest[:]), nil
}

func fingerprintTransaction(input ledger.TransactionInput) (transactionFingerprint, error) {
	template, err := stringAnnotation(input.Annotations(), ledger.AnnotationTransactionTemplateCode)
	if err != nil {
		return transactionFingerprint{}, err
	}

	direction, err := stringAnnotation(input.Annotations(), ledger.AnnotationTransactionDirection)
	if err != nil {
		return transactionFingerprint{}, err
	}

	annotations, err := fingerprintAnnotations(input.Annotations())
	if err != nil {
		return transactionFingerprint{}, err
	}

	entries := make([]entryFingerprint, 0, len(input.EntryInputs()))
	for entryIndex, entryInput := range input.EntryInputs() {
		if entryInput.PostingAddress() == nil {
			return transactionFingerprint{}, fmt.Errorf("entry %d posting address is nil", entryIndex)
		}

		route := entryInput.PostingAddress().Route().RoutingKey()
		entry := entryFingerprint{
			AccountType:    entryInput.PostingAddress().AccountType(),
			SubAccountID:   entryInput.PostingAddress().SubAccountID(),
			RouteKey:       route.Value(),
			RouteVersion:   route.Version(),
			Amount:         entryInput.Amount().String(),
			IdentityKey:    entryInput.IdentityKey(),
			SchemaVersion:  entryInput.SchemaVersion(),
			SourceChargeID: entryInput.SourceChargeID(),
			SpendChargeID:  entryInput.SpendChargeID(),
		}

		serialized, err := json.Marshal(entry)
		if err != nil {
			return transactionFingerprint{}, fmt.Errorf("serialize entry %d: %w", entryIndex, err)
		}
		entry.sortKey = string(serialized)
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].sortKey < entries[j].sortKey
	})

	return transactionFingerprint{
		BookedAt:    input.BookedAt().UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		Template:    template,
		Direction:   direction,
		Annotations: annotations,
		Entries:     entries,
	}, nil
}

func stringAnnotation(annotations models.Annotations, key string) (string, error) {
	value, ok := annotations[key]
	if !ok {
		return "", nil
	}

	stringValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("annotation %s must be a string", key)
	}

	return stringValue, nil
}

func fingerprintAnnotations(annotations models.Annotations) ([]fingerprintAnnotation, error) {
	output := make([]fingerprintAnnotation, 0, len(transactionFingerprintAnnotationKeys))

	for _, key := range transactionFingerprintAnnotationKeys {
		value, ok := annotations[key]
		if !ok {
			continue
		}

		serialized, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("serialize annotation %s: %w", key, err)
		}

		output = append(output, fingerprintAnnotation{
			Key:   key,
			Value: serialized,
		})
	}

	return output, nil
}
