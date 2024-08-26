package namespacedriver

import "context"

// NamespaceDecoder gets the namespace from the request.
type NamespaceDecoder interface {
	GetNamespace(ctx context.Context) (string, bool)
}

type StaticNamespaceDecoder string

func (d StaticNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool) {
	return string(d), true
}
