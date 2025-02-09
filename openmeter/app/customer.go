package app

type CustomerData interface {
	Validate() error
}

type CustomerApp struct {
	App          App
	CustomerData CustomerData
}
