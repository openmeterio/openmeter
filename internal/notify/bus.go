package notify

type Event interface {
	Payload() any

	Version() int // TODO: maybe semver?
	Type() string
}
