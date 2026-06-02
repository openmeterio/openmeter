package featuregate

type Gate interface {
	EvaluateBool(namespace, flag string, defaultValue bool) (bool, error)
}

func NewNoop() Gate {
	return Noop{}
}

type Noop struct{}

func (n Noop) EvaluateBool(string, string, bool) (bool, error) {
	return true, nil
}
