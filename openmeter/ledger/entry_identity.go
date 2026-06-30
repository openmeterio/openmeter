package ledger

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/equal"
)

type EntryIdentityVersion int

const (
	// legacy
	EntryIdentityVersion1 EntryIdentityVersion = 1
	// charge provenance
	EntryIdentityVersion2 EntryIdentityVersion = 2
)

func (e EntryIdentityVersion) prefix() string {
	if e == EntryIdentityVersion1 {
		return ""
	}

	return fmt.Sprintf("entry-identity:v%d:", e)
}

// EntryIdentityKeyText is the string serialized representation of the identity key. This encodes all information of the key.
type EntryIdentityKeyText string

func (e EntryIdentityKeyText) Version() EntryIdentityVersion {
	if strings.HasPrefix(string(e), EntryIdentityVersion2.prefix()) {
		return EntryIdentityVersion2
	}

	return EntryIdentityVersion1
}

func (e EntryIdentityKeyText) Parse() (EntryIdentityVersion, EntryIdentityParts, error) {
	version := e.Version()
	if version == EntryIdentityVersion1 {
		parts, err := parseV1EntryIdentityKey(string(e))
		if err != nil {
			return version, EntryIdentityParts{}, err
		}

		return version, parts.EntryIdentityParts(), nil
	}

	parts, err := parseV2EntryIdentityKey(string(e))
	if err != nil {
		return version, EntryIdentityParts{}, err
	}

	return version, parts.EntryIdentityParts(), nil
}

type EntryIdentityParts struct {
	CollectionSource *string // Custom key to keep 1:1 matching logic during collection
	CorrectionSource *string // References the ID of the entry we're collecting
	SourceChargeID   *string // The original creditpurchase charge (if exists) that funds this entry
	SpendChargeID    *string // The usage charge (if exists) that accrued this entry
}

func (e EntryIdentityParts) Text() (EntryIdentityKeyText, EntryIdentityVersion) {
	if e.SourceChargeID == nil && e.SpendChargeID == nil {
		return v1EntryIdentityParts{
			CollectionSource: e.CollectionSource,
			CorrectionSource: e.CorrectionSource,
		}.Text(), EntryIdentityVersion1
	}

	return v2EntryIdentityParts{
		CollectionSource: e.CollectionSource,
		CorrectionSource: e.CorrectionSource,
		SourceChargeID:   e.SourceChargeID,
		SpendChargeID:    e.SpendChargeID,
	}.Text(), EntryIdentityVersion2
}

func ValidateEntryIdentityKey(entry EntryInput) error {
	text := EntryIdentityKeyText(entry.IdentityKey())
	version, parts, err := text.Parse()
	if err != nil {
		return err
	}

	if (entry.SourceChargeID() != nil || entry.SpendChargeID() != nil) && version != EntryIdentityVersion2 {
		return fmt.Errorf("identity_key version must be %d when charge provenance is present", EntryIdentityVersion2)
	}

	if version == EntryIdentityVersion2 && parts.SourceChargeID == nil && parts.SpendChargeID == nil {
		return fmt.Errorf("identity_key version %d requires charge provenance", EntryIdentityVersion2)
	}

	if !equal.ComparablePtrEqual(parts.SourceChargeID, entry.SourceChargeID()) {
		return fmt.Errorf("source_charge_id does not match identity_key")
	}

	if !equal.ComparablePtrEqual(parts.SpendChargeID, entry.SpendChargeID()) {
		return fmt.Errorf("spend_charge_id does not match identity_key")
	}

	expected, expectedVersion := parts.Text()
	if version != expectedVersion || text != expected {
		return fmt.Errorf("identity_key is not canonical")
	}

	return nil
}

func NewCollectionSourceIdentityKey(index int) string {
	source := strconv.Itoa(index)
	text, _ := EntryIdentityParts{
		CollectionSource: &source,
	}.Text()

	return string(text)
}

func NewCorrectionSourceIdentityKey(sourceEntryID string) string {
	text, _ := EntryIdentityParts{
		CorrectionSource: &sourceEntryID,
	}.Text()

	return string(text)
}

type v1EntryIdentityParts struct {
	CollectionSource *string
	CorrectionSource *string
}

func (e v1EntryIdentityParts) Text() EntryIdentityKeyText {
	switch {
	case e.CollectionSource != nil:
		return EntryIdentityKeyText("collection-source:" + *e.CollectionSource)
	case e.CorrectionSource != nil:
		return EntryIdentityKeyText("correction-source:" + *e.CorrectionSource)
	default:
		return ""
	}
}

func (e v1EntryIdentityParts) EntryIdentityParts() EntryIdentityParts {
	return EntryIdentityParts{
		CollectionSource: e.CollectionSource,
		CorrectionSource: e.CorrectionSource,
	}
}

func parseV1EntryIdentityKey(identityKey string) (v1EntryIdentityParts, error) {
	switch {
	case strings.HasPrefix(identityKey, "collection-source:"):
		collectionSource := strings.TrimPrefix(identityKey, "collection-source:")

		return v1EntryIdentityParts{
			CollectionSource: &collectionSource,
		}, nil
	case strings.HasPrefix(identityKey, "correction-source:"):
		correctionSource := strings.TrimPrefix(identityKey, "correction-source:")

		return v1EntryIdentityParts{
			CorrectionSource: &correctionSource,
		}, nil
	default:
		return v1EntryIdentityParts{}, nil
	}
}

type v2EntryIdentityParts struct {
	CollectionSource *string
	CorrectionSource *string
	SourceChargeID   *string
	SpendChargeID    *string
}

func (e v2EntryIdentityParts) Text() EntryIdentityKeyText {
	return EntryIdentityKeyText(
		EntryIdentityVersion2.prefix() +
			escapeEntryIdentityPart(e.CollectionSource) + "|" +
			escapeEntryIdentityPart(e.CorrectionSource) + "|" +
			escapeEntryIdentityPart(e.SourceChargeID) + "|" +
			escapeEntryIdentityPart(e.SpendChargeID),
	)
}

func (e v2EntryIdentityParts) EntryIdentityParts() EntryIdentityParts {
	return EntryIdentityParts{
		CollectionSource: e.CollectionSource,
		CorrectionSource: e.CorrectionSource,
		SourceChargeID:   e.SourceChargeID,
		SpendChargeID:    e.SpendChargeID,
	}
}

func parseV2EntryIdentityKey(identityKey string) (v2EntryIdentityParts, error) {
	if !strings.HasPrefix(identityKey, EntryIdentityVersion2.prefix()) {
		return v2EntryIdentityParts{}, fmt.Errorf("identity_key version %d prefix is missing", EntryIdentityVersion2)
	}

	encoded := strings.TrimPrefix(identityKey, EntryIdentityVersion2.prefix())
	parts := strings.Split(encoded, "|")
	if len(parts) != 4 {
		return v2EntryIdentityParts{}, fmt.Errorf("invalid ledger entry identity key format")
	}

	collectionSource, err := parseOptionalEntryIdentityPart("collection_source", parts[0])
	if err != nil {
		return v2EntryIdentityParts{}, err
	}

	correctionSource, err := parseOptionalEntryIdentityPart("correction_source", parts[1])
	if err != nil {
		return v2EntryIdentityParts{}, err
	}

	sourceChargeID, err := parseOptionalEntryIdentityPart("source_charge_id", parts[2])
	if err != nil {
		return v2EntryIdentityParts{}, err
	}

	spendChargeID, err := parseOptionalEntryIdentityPart("spend_charge_id", parts[3])
	if err != nil {
		return v2EntryIdentityParts{}, err
	}

	return v2EntryIdentityParts{
		CollectionSource: collectionSource,
		CorrectionSource: correctionSource,
		SourceChargeID:   sourceChargeID,
		SpendChargeID:    spendChargeID,
	}, nil
}

func escapeEntryIdentityPart(value *string) string {
	return url.QueryEscape(lo.FromPtr(value))
}

func parseOptionalEntryIdentityPart(name, encoded string) (*string, error) {
	value, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", name, err)
	}

	if value == "" {
		return nil, nil
	}

	return lo.ToPtr(value), nil
}
