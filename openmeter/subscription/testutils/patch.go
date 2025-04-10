package subscriptiontestutils

import "github.com/openmeterio/openmeter/openmeter/subscription"

type TestPatch struct {
	PatchValue     any
	PatchOperation subscription.PatchOperation
	PatchPath      subscription.SpecPath

	ApplyToFn  func(s *subscription.SubscriptionSpec, c subscription.ApplyContext) error
	ValdiateFn func() error
}

var (
	_ subscription.Patch           = &TestPatch{}
	_ subscription.ValuePatch[any] = &TestPatch{}
)

func (p *TestPatch) ApplyTo(s *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
	if p.ApplyToFn != nil {
		return p.ApplyToFn(s, c)
	}
	return nil
}

func (p *TestPatch) Op() subscription.PatchOperation {
	return p.PatchOperation
}

func (p *TestPatch) Path() subscription.SpecPath {
	return p.PatchPath
}

func (p *TestPatch) Validate() error {
	if p.ValdiateFn != nil {
		return p.ValdiateFn()
	}
	return nil
}

func (p *TestPatch) MarshalJSON() ([]byte, error) {
	panic("not implemented")
}

func (p *TestPatch) UnmarshalJSON(data []byte) error {
	panic("not implemented")
}

func (p *TestPatch) Value() any {
	return p.PatchValue
}

func (p *TestPatch) ValueAsAny() any {
	return p.PatchValue
}
