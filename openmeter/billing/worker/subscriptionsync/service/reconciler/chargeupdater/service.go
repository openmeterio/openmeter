package chargeupdater

import "context"

type Updater interface {
	ApplyPatches(ctx context.Context, namespace string, patches []Patch) error
	LogPatches(patches []Patch)
}
