package annotationhook

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AnnotationCleanupHook struct {
	subscription.NoOpSubscriptionCommandHook
	subscriptionQueryService subscription.QueryService
	subscriptionRepo         subscription.SubscriptionRepository

	logger *slog.Logger
}

func NewAnnotationCleanupHook(subscriptionQueryService subscription.QueryService, subscriptionRepository subscription.SubscriptionRepository, logger *slog.Logger) (*AnnotationCleanupHook, error) {
	if subscriptionQueryService == nil {
		return nil, fmt.Errorf("subscription query service is required")
	}
	if subscriptionRepository == nil {
		return nil, fmt.Errorf("subscription repository is required")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &AnnotationCleanupHook{
		NoOpSubscriptionCommandHook: subscription.NoOpSubscriptionCommandHook{},
		subscriptionQueryService:    subscriptionQueryService,
		subscriptionRepo:            subscriptionRepository,
		logger:                      logger,
	}, nil
}

func (h *AnnotationCleanupHook) BeforeDelete(ctx context.Context, view subscription.SubscriptionView) error {
	if err := h.updateSupersedingSubscriptionAnnotations(ctx, view); err != nil {
		return fmt.Errorf("failed to update superseding subscription annotations: %w", err)
	}
	if err := h.updatePreviousSubscriptionAnnotations(ctx, view); err != nil {
		return fmt.Errorf("failed to update previous subscription annotations: %w", err)
	}
	return nil
}

func (h *AnnotationCleanupHook) updateSupersedingSubscriptionAnnotations(ctx context.Context, view subscription.SubscriptionView) error {
	supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(view.Subscription.Annotations)
	previousID := subscription.AnnotationParser.GetPreviousSubscriptionID(view.Subscription.Annotations)

	if supersedingID == nil {
		return nil
	}

	supersedingView, err := h.subscriptionQueryService.GetView(ctx, models.NamespacedID{
		ID:        lo.FromPtr(supersedingID),
		Namespace: view.Subscription.Namespace,
	})
	if err != nil {
		if subscription.IsSubscriptionNotFoundError(err) {
			h.logger.Error("superseding subscription not found, continuing without cleanup",
				"error", err,
				"supersedingID", lo.FromPtr(supersedingID),
				"previousID", lo.FromPtr(previousID),
				"subscription", view.Subscription,
			)

			return nil
		}

		return fmt.Errorf("failed to get superseding subscription: %w", err)
	}

	supersedingAnnotations := supersedingView.Subscription.Annotations
	if supersedingAnnotations != nil {
		supersedingAnnotations = maps.Clone(supersedingAnnotations)
	} else {
		supersedingAnnotations = models.Annotations{}
	}

	// If the deleted subscription had a previous subscription, link the superseding to it
	// This is a safety behavior, as
	// - were multiple scheduled subscriptions allowed this would keep them linked together
	if previousID != nil {
		supersedingAnnotations, err = subscription.AnnotationParser.SetPreviousSubscriptionID(supersedingAnnotations, *previousID)
		if err != nil {
			return fmt.Errorf("failed to update superseding subscription's previous ID: %w", err)
		}
		_, err = h.subscriptionRepo.UpdateAnnotations(ctx, supersedingView.Subscription.NamespacedID, supersedingAnnotations)
		if err != nil {
			return fmt.Errorf("failed to update superseding subscription annotations: %w", err)
		}
	} else {
		// Otherwise, clear the previous subscription ID from the superseding subscription
		if supersedingAnnotations == nil {
			// Nothing to clear if annotations are nil, skip update
			return nil
		}
		delete(supersedingAnnotations, subscription.AnnotationPreviousSubscriptionID)
		// If the map is now empty, set it to nil
		if len(supersedingAnnotations) == 0 {
			supersedingAnnotations = nil
		}
		_, err = h.subscriptionRepo.UpdateAnnotations(ctx, supersedingView.Subscription.NamespacedID, supersedingAnnotations)
		if err != nil {
			return fmt.Errorf("failed to update superseding subscription annotations: %w", err)
		}
	}

	return nil
}

func (h *AnnotationCleanupHook) updatePreviousSubscriptionAnnotations(ctx context.Context, view subscription.SubscriptionView) error {
	supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(view.Subscription.Annotations)
	previousID := subscription.AnnotationParser.GetPreviousSubscriptionID(view.Subscription.Annotations)

	if previousID == nil {
		return nil
	}

	previousView, err := h.subscriptionQueryService.GetView(ctx, models.NamespacedID{
		ID:        lo.FromPtr(previousID),
		Namespace: view.Subscription.Namespace,
	})
	if err != nil {
		if subscription.IsSubscriptionNotFoundError(err) {
			h.logger.Error("previous subscription not found, continuing without cleanup",
				"error", err,
				"supersedingID", lo.FromPtr(supersedingID),
				"previousID", lo.FromPtr(previousID),
				"subscription", view.Subscription,
			)

			return nil
		}

		return fmt.Errorf("failed to get previous subscription: %w", err)
	}

	previousAnnotations := previousView.Subscription.Annotations
	if previousAnnotations != nil {
		previousAnnotations = maps.Clone(previousAnnotations)
	} else {
		previousAnnotations = models.Annotations{}
	}

	// If the deleted subscription had a superseding subscription, link the previous to it
	if supersedingID != nil {
		previousAnnotations, err = subscription.AnnotationParser.SetSupersedingSubscriptionID(previousAnnotations, *supersedingID)
		if err != nil {
			return fmt.Errorf("failed to update previous subscription's superseding ID: %w", err)
		}
		_, err = h.subscriptionRepo.UpdateAnnotations(ctx, previousView.Subscription.NamespacedID, previousAnnotations)
		if err != nil {
			return fmt.Errorf("failed to update previous subscription annotations: %w", err)
		}
	} else {
		// Otherwise, clear the superseding subscription ID from the previous subscription
		if previousAnnotations == nil {
			// Nothing to clear if annotations are nil, skip update
			return nil
		}
		previousAnnotations, err = subscription.AnnotationParser.ClearSupersedingSubscriptionID(previousAnnotations)
		if err != nil {
			return fmt.Errorf("failed to clear previous subscription's superseding ID: %w", err)
		}
		// If the map is now empty, set it to nil
		if len(previousAnnotations) == 0 {
			previousAnnotations = nil
		}
		_, err = h.subscriptionRepo.UpdateAnnotations(ctx, previousView.Subscription.NamespacedID, previousAnnotations)
		if err != nil {
			return fmt.Errorf("failed to update previous subscription annotations: %w", err)
		}
	}

	return nil
}
