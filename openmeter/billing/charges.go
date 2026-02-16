package billing

import "errors"

type SetChargeIDsOnInvoiceLinesInput struct {
	Namespace        string
	LineIDToChargeID map[string]string
}

func (i SetChargeIDsOnInvoiceLinesInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.LineIDToChargeID == nil {
		return errors.New("line id to charge id map is required")
	}

	return nil
}

type SetChargeIDsOnSplitlineGroupsInput struct {
	Namespace         string
	GroupIDToChargeID map[string]string
}

func (i SetChargeIDsOnSplitlineGroupsInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.GroupIDToChargeID == nil {
		return errors.New("group id to charge id map is required")
	}

	return nil
}
