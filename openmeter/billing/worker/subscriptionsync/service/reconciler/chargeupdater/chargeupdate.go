package chargeupdater

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type updater struct {
	chargesService charges.Service
	logger         *slog.Logger
}

func New(chargesService charges.Service, logger *slog.Logger) Updater {
	return &updater{
		chargesService: chargesService,
		logger:         logger,
	}
}

func (u *updater) ApplyPatches(ctx context.Context, namespace string, patches []Patch) error {
	if u.chargesService == nil {
		return fmt.Errorf("charges service is required")
	}

	parsed, err := u.parsePatches(patches)
	if err != nil {
		return fmt.Errorf("parsing patches: %w", err)
	}

	if len(parsed.creates) == 0 {
		return nil
	}

	_, err = u.chargesService.Create(ctx, charges.CreateInput{
		Namespace: namespace,
		Intents:   parsed.creates,
	})
	if err != nil {
		return fmt.Errorf("creating charges [namespace=%s]: %w", namespace, err)
	}

	return nil
}

func (u *updater) LogPatches(patches []Patch) {
	for _, patch := range patches {
		patch.Log(u.logger)
	}
}

type patchesParsed struct {
	creates charges.ChargeIntents
}

func (u *updater) parsePatches(patches []Patch) (patchesParsed, error) {
	parsed := patchesParsed{}

	for _, patch := range patches {
		switch patch.Op() {
		case PatchOpCreate:
			createPatch, err := patch.AsCreatePatch()
			if err != nil {
				return patchesParsed{}, fmt.Errorf("getting create patch: %w", err)
			}

			parsed.creates = append(parsed.creates, createPatch.Intent)
		default:
			return patchesParsed{}, fmt.Errorf("unexpected patch operation: %s", patch.Op())
		}
	}

	return parsed, nil
}
