package stddetailedline

// GODEBUG=gotypesalias=0 keeps type aliases transparent for goderive's go/types
// analysis (Go 1.22+ types.Alias nodes this goderive version does not unwrap).
// See openmeter/billing/generate.go for the full rationale.
//go:generate env GODEBUG=gotypesalias=0 go run github.com/awalterschulze/goderive
