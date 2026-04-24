# operation

<!-- archie:ai-start -->

> Defines the Operation[Request,Response] function type — the single callable unit that transport layers (httptransport.Handler) invoke — plus Middleware[Req,Resp], Chain, Compose, and AsNoResponseOperation helpers for building and composing RPC-style operations.

## Patterns

**Operation is a plain function type** — type Operation[Request any, Response any] func(ctx context.Context, request Request) (Response, error) — no interface, no struct. Assign any matching func literal directly. (`var op operation.Operation[MyReq, MyResp] = myServiceMethod`)
**Middleware wraps Operation** — type Middleware[Req,Resp] func(Operation[Req,Resp]) Operation[Req,Resp]. Chain(outer, ...others) composes them outermost-first. (`chain := operation.Chain[Req, Resp](authMW, loggingMW); wrapped := chain(myOp)`)
**AsNoResponseOperation for void-returning handlers** — Wraps func(ctx, Request) error as Operation[Request, any] so Delete-style handlers can use the same middleware chain. (`op := operation.AsNoResponseOperation(func(ctx context.Context, req DeleteReq) error { ... })`)
**Compose pipelines two operations** — Compose(op1, op2) produces Operation[Req,Resp] where op1's response becomes op2's request — use for get-then-update patterns. (`combined := operation.Compose(getOp, updateOp)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `operation.go` | Core type definitions: Operation, AsNoResponseOperation, Compose. | No context passed to AsNoResponseOperation's inner func — the outer ctx flows through naturally. |
| `middleware.go` | Middleware and Chain; middlewares execute in declaration order (outermost = first declared). | Chain reverses the slice internally (for i := len(others)-1; i >= 0; i--) so the first argument to Chain is always the outermost. |

## Anti-Patterns

- Implementing business logic inside a Middleware — middleware is for cross-cutting concerns (auth, logging, tracing) only
- Returning a non-nil Response on error — callers may use zero-value Response alongside errors; Compose propagates errors correctly only when Response is zero on error

## Decisions

- **Function type instead of interface** — A function type requires no method sets and avoids boilerplate; any matching func can be used directly without wrapping, while still being chainable via Middleware.

<!-- archie:ai-end -->
