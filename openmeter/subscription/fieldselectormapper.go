package subscription

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/treex"
)

type fieldMappingSelector string

const (
	PhaseSelector fieldMappingSelector = "phase"
)

// MapSubscriptionSpecValidationIssueFieldSelectors maps the FieldSelectors of a ValidationIssue from the structure of SubscriptionSpec
// to the structure of api.SubscriptionView
func MapSubscriptionSpecValidationIssueFieldSelectors(iss models.ValidationIssue) (models.ValidationIssue, error) {
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

			if attrs := desc.GetAttributes(); attrs != nil {
				if _, ok := attrs[PhaseSelector]; ok {
					// We've found a phase selector, it is expected to have two leafes which we'll combine to a single descriptor with expressions and then swap it

					// Let's get the leafs from a tree starting at this node
					t2, err := treex.NewTree(n)
					if err != nil {
						return false, err
					}

					leafs := t2.Leafs()
					if len(leafs) != 2 {
						return false, fmt.Errorf("phase selector segment %s has %d leafs, expected 2", desc.String(), len(leafs))
					}

					phaseKey := lo.ToPtr(lo.FromPtr(leafs[1].Value())).String()

					prunedAttrs := attrs.Clone()
					delete(prunedAttrs, PhaseSelector)

					mappedDesc := models.NewFieldSelector("phases").
						WithExpression(models.NewFieldAttrValue("key", phaseKey)).
						WithAttributes(prunedAttrs)

					// Now we'll swap desc with mappedDesc. We don't need to walk any further as this is the only mapping we'll do.
					return true, t.Swap(n, mappedDesc)
				}
			}

			return false, nil
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
