package price

type Repository interface {
	GetByID(id string) (*Price, error)
	Get(subscriptionId string, phaseKey string, itemKey string) (*Price, error)
	Create(input CreateInput) (*Price, error)
}
