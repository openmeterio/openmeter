package subscription

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/treex"
)

type fieldDescriptorMapping string

const (
	PhaseDescriptor fieldDescriptorMapping = "phase"
)

// MapSubscriptionSpecValidationIssueField maps the FieldSelectors of a ValidationIssue
// from the structure of SubscriptionSpec to the structure of api.SubscriptionView
func MapSubscriptionSpecValidationIssueField(iss models.ValidationIssue) (models.ValidationIssue, error) {
	// We'll do a tree walk and if we see annotated nodes we swap them with their mapping
	var mappedField *models.FieldDescriptor
	field := iss.Field()

	if field == nil {
		return iss, nil
	}

	err := field.Tree(func(t *models.FieldDescriptorTree) error {
		err := t.DFS(func(n *treex.Node[*models.FieldDescriptor]) (bool, error) {
			desc := n.Value()

			// this will not happen for valid field descriptors but let's guard against nil anyways
			if desc == nil {
				return false, errors.New("field descriptor is nil")
			}

			return mapPhaseDescriptor(t, desc)
		})
		if err != nil {
			return err
		}

		// We've finished the walk, we'll update the mapped field to be the tree root (as that might have been changed by the mapping)
		mappedField = t.Root().Value().Clone()

		return nil
	})
	if err != nil {
		return iss, err
	}

	return iss.WithField(mappedField), nil
}

// mapPhaseDescriptor swaps a subtree to a single node with required FieldExpressions
func mapPhaseDescriptor(t *models.FieldDescriptorTree, current *models.FieldDescriptor) (bool, error) {
	if attrs := current.GetAttributes(); attrs != nil {
		if _, ok := attrs[PhaseDescriptor]; ok {
			// Let's get the leafs from a tree starting at this node
			leafs := make([]*models.FieldDescriptor, 0)

			err := current.Tree(func(t *models.FieldDescriptorTree) error {
				leafs = t.Leafs()

				return nil
			})
			if err != nil {
				return false, err
			}

			if len(leafs) != 2 {
				return false, fmt.Errorf("phase selector segment %s has %d leafs, expected 2", current.String(), len(leafs))
			}

			phaseKey := leafs[1].String()

			prunedAttrs := attrs.Clone()
			delete(prunedAttrs, PhaseDescriptor)

			mappedDesc := models.NewFieldSelector("phases").
				WithExpression(models.NewFieldAttrValue("key", phaseKey)).
				WithAttributes(prunedAttrs)

			// Now we'll swap desc with mappedDesc. We don't need to walk any further as this is the only mapping we'll do.
			return true, t.Swap(current, mappedDesc)
		}
	}

	return false, nil
}
