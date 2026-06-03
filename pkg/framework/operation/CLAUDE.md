# operation

<!-- archie:ai-start -->

> Defines the Operation[Request,Response] function type — the single callable unit transport layers (httptransport.Handler) invoke — plus Middleware[Req,Resp], Chain, Compose, and AsNoResponseOperation helpers for building and composing RPC-style operations.

## Patterns

**Operation is a plain function type** — type Operation[Request any, Response any] func(ctx context.Context, request Request) (Response, error). No interface, no wrapper struct — assign any matching func literal or method reference directly. (`var op operation.Operation[MyReq, MyResp] = myServiceMethod`)
**Middleware wraps Operation — Chain composes outermost-first** — type Middleware[Req,Resp] func(Operation[Req,Resp]) Operation[Req,Resp]. Chain(outer, others...) reverses others internally so the first argument is always executed outermost (first on the request path). (`chain := operation.Chain[Req, Resp](authMW, loggingMW); wrapped := chain(myOp)`)
**AsNoResponseOperation for void/Delete-style handlers** — Wraps func(ctx, Request) error as Operation[Request, any] so Delete-style handlers reuse the same middleware chain and httptransport.Handler pipeline without returning a body. (`op := operation.AsNoResponseOperation(func(ctx context.Context, req DeleteReq) error { return svc.Delete(ctx, req.ID) })`)
**Compose pipelines two operations** — Compose(op1, op2) yields Operation[Req,Resp] where op1's response becomes op2's request. On op1 error, op2 is never called and the zero-value Response is returned. (`combined := operation.Compose(getOp, updateOp)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `operation.go` | Core type definitions: Operation, AsNoResponseOperation, Compose. | AsNoResponseOperation returns Operation[Request, any] — the response type is always any/nil. Compose propagates errors correctly only when op1 returns the zero-value Response alongside the error. |
| `middleware.go` | Middleware and Chain; middlewares execute in declaration order (outermost = first declared). | Chain reverses the others slice internally (for i := len(others)-1; i >= 0; i--) so the first argument is always the outermost wrapper. |

## Anti-Patterns

- Implementing business logic inside a Middleware — middleware is for cross-cutting concerns (auth, logging, tracing) only.
- Returning a non-nil Response on error from an Operation — Compose propagates errors correctly only when Response is zero-valued alongside the error.
- Wrapping the same operation in Chain twice with the same middleware — the middleware executes twice on the hot path.

## Decisions

- **Function type instead of interface.** — A function type needs no method sets and avoids boilerplate; any matching func is usable directly while still chainable via Middleware and compatible with httptransport.Handler generics.

## Example: Wrap a service method as an Operation with middleware for httptransport.Handler

```
import "github.com/openmeterio/openmeter/pkg/framework/operation"

var op operation.Operation[ListReq, []Item] = svc.List
withAuth := operation.Chain[ListReq, []Item](authMiddleware, loggingMiddleware)
protected := withAuth(op)
delOp := operation.AsNoResponseOperation(func(ctx context.Context, req DeleteReq) error {
    return svc.Delete(ctx, req.ID)
})
```

<!-- archie:ai-end -->
