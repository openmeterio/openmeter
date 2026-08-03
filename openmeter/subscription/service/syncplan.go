package service

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type syncAction string

const (
	syncActionArchive syncAction = "archive"
	syncActionCreate  syncAction = "create"
)

type syncCommand struct {
	action syncAction
	node   syncNode
}

func (c syncCommand) rank() int {
	if c.action == syncActionArchive {
		// Archive children before their parents.
		switch c.node.Ref().Kind {
		case syncNodeKindEntitlement:
			return 0
		case syncNodeKindItem:
			return 1
		case syncNodeKindPhase:
			return 2
		}
	}

	// Create parents before their children. Entitlements precede items because
	// the item stores the entitlement reference.
	switch c.node.Ref().Kind {
	case syncNodeKindPhase:
		return 0
	case syncNodeKindEntitlement:
		return 1
	case syncNodeKindItem:
		return 2
	default:
		return 3
	}
}

type syncPlan struct {
	commands []syncCommand
	refState syncReferences
}

func (p syncPlan) Execute(ctx context.Context, service *service, customer customer.Customer) error {
	execution := &syncExecution{
		service:  service,
		customer: customer,
		refState: syncReferences{
			phases:       maps.Clone(p.refState.phases),
			entitlements: maps.Clone(p.refState.entitlements),
		},
	}

	for _, command := range p.commands {
		ref := command.node.Ref()

		var err error
		switch command.action {
		case syncActionArchive:
			err = command.node.Archive(ctx, execution)
		case syncActionCreate:
			err = command.node.Create(ctx, execution)
		default:
			err = fmt.Errorf("unknown sync action %q", command.action)
		}
		if err != nil {
			return fmt.Errorf(
				"failed to %s %s at %s: %w",
				command.action,
				ref.Kind,
				ref.Path,
				err,
			)
		}
	}

	return nil
}

func newSyncPlan(
	view subscription.SubscriptionView,
	desiredSpec subscription.SubscriptionSpec,
) (syncPlan, error) {
	nodes, refState, err := buildSyncNodes(view, desiredSpec)
	if err != nil {
		return syncPlan{}, err
	}

	changes := make(map[syncNodeRef]syncNodeChange, len(nodes))
	for ref, node := range nodes {
		change, err := node.Compare()
		if err != nil {
			return syncPlan{}, fmt.Errorf("failed to compare %s at %s: %w", ref.Kind, ref.Path, err)
		}
		changes[ref] = change
	}

	// A replaced parent invalidates its descendants' persisted references.
	propagatePhaseChanges(nodes, changes)
	// Items persist entitlement references, so either changing replaces both.
	propagateItemEntitlementChanges(nodes, changes)

	commands := make([]syncCommand, 0, len(changes)*2)
	for ref, change := range changes {
		node := nodes[ref]
		switch change {
		case syncNodeChangeCreate:
			commands = append(commands, syncCommand{action: syncActionCreate, node: node})
		case syncNodeChangeArchive:
			commands = append(commands, syncCommand{action: syncActionArchive, node: node})
		case syncNodeChangeReplace:
			commands = append(commands,
				syncCommand{action: syncActionArchive, node: node},
				syncCommand{action: syncActionCreate, node: node},
			)
		}
	}

	sortSyncCommands(commands)

	return syncPlan{
		commands: commands,
		refState: refState,
	}, nil
}

func buildSyncNodes(
	view subscription.SubscriptionView,
	desiredSpec subscription.SubscriptionSpec,
) (map[syncNodeRef]syncNode, syncReferences, error) {
	nodes := map[syncNodeRef]syncNode{}
	refState := syncReferences{
		phases:       map[subscription.SpecPath]subscription.SubscriptionPhase{},
		entitlements: map[subscription.SpecPath]*subscription.SubscriptionEntitlement{},
	}
	currentPhases := make(map[string]*subscription.SubscriptionPhaseView, len(view.Phases))
	for i := range view.Phases {
		phase := &view.Phases[i]
		currentPhases[phase.SubscriptionPhase.Key] = phase
	}

	phaseKeys := make(map[string]struct{}, len(currentPhases)+len(desiredSpec.Phases))
	for key := range currentPhases {
		phaseKeys[key] = struct{}{}
	}
	for key := range desiredSpec.Phases {
		phaseKeys[key] = struct{}{}
	}

	for phaseKey := range phaseKeys {
		currentPhase := currentPhases[phaseKey]
		desiredPhase := desiredSpec.Phases[phaseKey]
		phasePath := subscription.NewPhasePath(phaseKey)
		phaseRef := syncNodeRef{Path: phasePath, Kind: syncNodeKindPhase}

		var currentCadence models.CadencedModel
		if currentPhase != nil {
			var err error
			currentCadence, err = view.Spec.GetPhaseCadence(phaseKey)
			if err != nil {
				return nil, syncReferences{}, fmt.Errorf("failed to get current cadence for phase %s: %w", phaseKey, err)
			}
			refState.phases[phasePath] = currentPhase.SubscriptionPhase
		}

		var desiredCadence models.CadencedModel
		if desiredPhase != nil {
			var err error
			desiredCadence, err = desiredSpec.GetPhaseCadence(phaseKey)
			if err != nil {
				return nil, syncReferences{}, fmt.Errorf("failed to get desired cadence for phase %s: %w", phaseKey, err)
			}
		}

		nodes[phaseRef] = &phaseSyncNode{
			ref:            phaseRef,
			subscription:   view.Subscription,
			current:        currentPhase,
			desired:        desiredPhase,
			desiredCadence: desiredCadence,
		}

		itemKeys := map[string]struct{}{}
		if currentPhase != nil {
			for key := range currentPhase.ItemsByKey {
				itemKeys[key] = struct{}{}
			}
		}
		if desiredPhase != nil {
			for key := range desiredPhase.ItemsByKey {
				itemKeys[key] = struct{}{}
			}
		}

		for itemKey := range itemKeys {
			var currentItems []subscription.SubscriptionItemView
			if currentPhase != nil {
				currentItems = currentPhase.ItemsByKey[itemKey]
			}

			var desiredItems []*subscription.SubscriptionItemSpec
			if desiredPhase != nil {
				desiredItems = desiredPhase.ItemsByKey[itemKey]
			}

			itemCount := max(len(currentItems), len(desiredItems))
			for itemIdx := range itemCount {
				var currentItem *subscription.SubscriptionItemView
				if itemIdx < len(currentItems) {
					currentItem = &currentItems[itemIdx]
				}

				var desiredItem *subscription.SubscriptionItemSpec
				if itemIdx < len(desiredItems) {
					desiredItem = desiredItems[itemIdx]
				}

				itemPath := subscription.NewItemVersionPath(phaseKey, itemKey, itemIdx)
				itemRef := syncNodeRef{Path: itemPath, Kind: syncNodeKindItem}
				nodes[itemRef] = &itemSyncNode{
					ref:                 itemRef,
					phasePath:           phasePath,
					subscription:        view.Subscription,
					current:             currentItem,
					desired:             desiredItem,
					currentPhaseCadence: currentCadence,
					desiredPhaseCadence: desiredCadence,
				}

				var currentEntitlement *subscription.SubscriptionEntitlement
				if currentItem != nil {
					currentEntitlement = currentItem.Entitlement
				}

				var desiredEntitlement *subscription.ScheduleSubscriptionEntitlementInput
				if desiredItem != nil {
					input, hasEntitlement, err := getItemEntitlementInput(createItemOptions{
						cust:         view.Customer,
						sub:          view.Subscription,
						phaseCadence: desiredCadence,
						itemSpec:     *desiredItem,
					})
					if err != nil {
						return nil, syncReferences{}, fmt.Errorf("failed to derive entitlement for item %s: %w", itemPath, err)
					}
					if hasEntitlement {
						desiredEntitlement = &input
					}
				}

				if currentEntitlement != nil || desiredEntitlement != nil {
					entitlementRef := syncNodeRef{Path: itemPath, Kind: syncNodeKindEntitlement}
					nodes[entitlementRef] = &entitlementSyncNode{
						ref:          entitlementRef,
						subscription: view.Subscription,
						currentItem:  currentItem,
						current:      currentEntitlement,
						desired:      desiredEntitlement,
					}
					if currentEntitlement != nil {
						refState.entitlements[itemPath] = currentEntitlement
					}
				}
			}
		}
	}

	return nodes, refState, nil
}

func propagatePhaseChanges(
	nodes map[syncNodeRef]syncNode,
	changes map[syncNodeRef]syncNodeChange,
) {
	for ref, change := range changes {
		if ref.Kind != syncNodeKindPhase || change != syncNodeChangeReplace {
			continue
		}

		for childRef, child := range nodes {
			if childRef == ref || !ref.Path.IsParentOf(childRef.Path) {
				continue
			}
			changes[childRef] = child.forceChange()
		}
	}
}

func propagateItemEntitlementChanges(
	nodes map[syncNodeRef]syncNode,
	changes map[syncNodeRef]syncNodeChange,
) {
	for ref := range nodes {
		if ref.Kind != syncNodeKindItem {
			continue
		}

		entitlementRef := syncNodeRef{Path: ref.Path, Kind: syncNodeKindEntitlement}
		entitlementNode, hasEntitlement := nodes[entitlementRef]
		if !hasEntitlement {
			continue
		}

		if changes[ref] == syncNodeChangeUnchanged && changes[entitlementRef] == syncNodeChangeUnchanged {
			continue
		}

		changes[ref] = nodes[ref].forceChange()
		changes[entitlementRef] = entitlementNode.forceChange()
	}
}

func sortSyncCommands(commands []syncCommand) {
	slices.SortStableFunc(commands, func(left, right syncCommand) int {
		if left.action != right.action {
			if left.action == syncActionArchive {
				return -1
			}

			return 1
		}

		if rank := cmp.Compare(left.rank(), right.rank()); rank != 0 {
			return rank
		}

		leftRef := left.node.Ref()
		rightRef := right.node.Ref()
		if path := cmp.Compare(leftRef.Path, rightRef.Path); path != 0 {
			return path
		}

		return cmp.Compare(leftRef.Kind, rightRef.Kind)
	})
}

type syncReferences struct {
	phases       map[subscription.SpecPath]subscription.SubscriptionPhase
	entitlements map[subscription.SpecPath]*subscription.SubscriptionEntitlement
}

// syncExecution is mutable state for reference tracking during an update.
type syncExecution struct {
	// service applies the persistence operations represented by commands.
	service *service
	// customer supplies customer data needed when materializing items.
	customer customer.Customer
	// refState tracks the latest materialized phase and entitlement references.
	refState syncReferences
}
