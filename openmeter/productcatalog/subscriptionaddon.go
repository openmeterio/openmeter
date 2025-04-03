package productcatalog

// TODO: change to AddonInstanceType once defined
type SubscriptionAddonInstanceType string

const (
	SubscriptionAddonInstanceTypeSingle   SubscriptionAddonInstanceType = "single"
	SubscriptionAddonInstanceTypeMultiple SubscriptionAddonInstanceType = "multiple"
)

func (s SubscriptionAddonInstanceType) StringValues() []string {
	return []string{
		string(SubscriptionAddonInstanceTypeSingle),
		string(SubscriptionAddonInstanceTypeMultiple),
	}
}
