package chargeupdater

import "context"

// Updater acts as a facade for the charges service. This might be a temporary solution as charges will
// have it's own patching functionality in the future.
//
// For now we are keeping this until charges patches is fully operational.
type Updater interface {
	ApplyPatches(ctx context.Context, namespace string, patches []Patch) error
	LogPatches(patches []Patch)
}
