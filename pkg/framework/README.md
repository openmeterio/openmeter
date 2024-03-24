# Framework

## Implementing a new operation

An _operation_ is a single method or function that a caller can invoke. In an HTTP API context it's often called a _route_ or _endpoint_.

This framework promotes a bottom-up approach to building APIs.

The first step to implementing a new operation is to define the request and response types along with an operation function.

```go
type Request struct {
    // request params
}

type Response struct {
    // response data
}

func Operation(ctx context.Context, req Request) (Response, error) {
    // operation logic
}
```

Alternatively, the operation function can be defined as a method on a struct.

```go
type Service struct {

}

func (s Service) Operation(ctx context.Context, req Request) (Response, error) {
    // operation logic
}
```

In case of an HTTP API, the next step is to define encoding and decoding functions for request, response and errors.

```go
func DecodeOperationRequest(ctx context.Context, r *http.Request) (Request, error) {
	// decode request
}

func EncodeOperationResponse(ctx context.Context, w http.ResponseWriter, response Response) error {
	// encode response
}

func EncodeOperationError(ctx context.Context, err error, w http.ResponseWriter) bool {
	// encode error

  // return true if the error is considered "handled", false otherwise (error gets passed to the error handler)
  return true
}
```

Finally, create a constructor function for the HTTP handler.

```go
func NewOperationHandler(op operation.Operation[Request, Response], errorHandler httptransport.ErrorHandler) http.Handler {
	return httptransport.NewHandler(
		op,
		DecodeOperationRequest,
		EncodeOperationResponse,
		EncodeOperationError,
		httptransport.WithErrorHandler(errorHandler),
		httptransport.WithOperationName("operation"),
	)
}
```

Alternatively, the constructor function can instantiate the operation itself as well.

```go
func NewOperationHandler(errorHandler httptransport.ErrorHandler) http.Handler {
	return httptransport.NewHandler(
		NewOperation(),
		DecodeOperationRequest,
		EncodeOperationResponse,
		EncodeOperationError,
		httptransport.WithErrorHandler(errorHandler),
		httptransport.WithOperationName("operation"),
	)
}
```

Register the HTTP handler in the router.
