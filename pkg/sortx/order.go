package sortx

type Order string

const (
	OrderAsc     Order = "ASC"
	OrderDesc    Order = "DESC"
	OrderDefault Order = OrderAsc
	OrderNone    Order = ""
)

func (s Order) String() string {
	return string(s)
}

// TODO (andras): name is misleading as it checks if the order is not set rather than if it's the default value
func (s Order) IsDefaultValue() bool {
	return s == OrderNone
}
