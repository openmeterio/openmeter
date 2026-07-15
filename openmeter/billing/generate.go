package billing

// GODEBUG=gotypesalias=0 keeps type aliases transparent for goderive's go/types
// analysis. On Go 1.22+ an alias (e.g. productcatalog.UnitConfig, which aliases
// unitconfig.UnitConfig) is a first-class types.Alias node that this goderive
// version does not unwrap; without the flag it fails to see the aliased type's
// hand-written Equal method, auto-derives equality instead, and recurses into
// alpacadecimal.Decimal until it OOMs. Remove once goderive unwraps aliases.
//go:generate env GODEBUG=gotypesalias=0 go run github.com/awalterschulze/goderive
