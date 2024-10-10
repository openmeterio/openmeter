package applieddiscount

type Repository interface {
	GetByID(id string) (*AppliedDiscount, error)
	Get(subscriptionID string, phaseKey string) ([]*AppliedDiscount, error)
	Create(input CreateInput) (*AppliedDiscount, error)
}
