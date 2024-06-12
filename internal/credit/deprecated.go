package credit

// things in here we are trying to delete

type DELETEME_EntryType string

// Defines values for EntryType.
const (
	EntryTypeGrant     DELETEME_EntryType = "GRANT"
	EntryTypeVoidGrant DELETEME_EntryType = "VOID_GRANT"
	EntryTypeReset     DELETEME_EntryType = "RESET"
)

func (DELETEME_EntryType) Values() (kinds []string) {
	for _, s := range []DELETEME_EntryType{
		EntryTypeGrant,
		EntryTypeVoidGrant,
		EntryTypeReset,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

type DELETEME_GrantType string

// Defines values for GrantType.
const (
	GrantTypeUsage DELETEME_GrantType = "USAGE"
)

func (DELETEME_GrantType) Values() (kinds []string) {
	for _, s := range []DELETEME_GrantType{
		GrantTypeUsage,
	} {
		kinds = append(kinds, string(s))
	}
	return
}
