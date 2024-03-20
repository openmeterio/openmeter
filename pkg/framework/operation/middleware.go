package operation

// Middleware is a chainable function that wraps an [Operation].
type Middleware[Request any, Response any] func(Operation[Request, Response]) Operation[Request, Response]

// Chain is a helper function for composing middlewares. Requests will
// traverse them in the order they're declared. That is, the first middleware
// is treated as the outermost middleware.
func Chain[Request any, Response any](outer Middleware[Request, Response], others ...Middleware[Request, Response]) Middleware[Request, Response] {
	return func(next Operation[Request, Response]) Operation[Request, Response] {
		for i := len(others) - 1; i >= 0; i-- { // reverse
			next = others[i](next)
		}
		return outer(next)
	}
}
