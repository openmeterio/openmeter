package transaction

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
)

type postCommitCallbacksKey struct{}

// postCommitCallbacks collects callbacks to run after the outermost transaction commits.
// Stored as a pointer in context so mutations are visible across the entire context chain.
type postCommitCallbacks struct {
	callbacks []func(ctx context.Context)
}

// OnCommit registers a callback to run after the outermost transaction commits successfully.
// If called outside a managed transaction, the callback is executed immediately.
func OnCommit(ctx context.Context, fn func(ctx context.Context)) {
	cbs, ok := ctx.Value(postCommitCallbacksKey{}).(*postCommitCallbacks)
	if !ok || cbs == nil {
		// Not inside a managed transaction — execute immediately.
		fn(ctx)
		return
	}
	cbs.callbacks = append(cbs.callbacks, fn)
}

func initPostCommitCallbacks(ctx context.Context) context.Context {
	return context.WithValue(ctx, postCommitCallbacksKey{}, &postCommitCallbacks{})
}

func runPostCommitCallbacks(ctx context.Context) {
	cbs, ok := ctx.Value(postCommitCallbacksKey{}).(*postCommitCallbacks)
	if !ok || cbs == nil {
		return
	}
	for _, fn := range cbs.callbacks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic in post-commit callback", "error", fmt.Sprintf("%v:\n%s", r, debug.Stack()))
				}
			}()
			fn(ctx)
		}()
	}
}
