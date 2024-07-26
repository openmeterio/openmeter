package sortx

type Order string

const (
	OrderAsc     Order = "ASC"
	OrderDesc    Order = "DESC"
	OrderDefault Order = "ASC"
	OrderNone    Order = ""
)

func (s Order) String() string {
	return string(s)
}

func (s Order) IsDefaultValue() bool {
	return s == ""
}
