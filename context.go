package geordi

import (
	"context"
	"time"

	"github.com/benbjohnson/clock"
)

type processKey struct{}

// Self gets the enclosing Process from the context.
func Self(ctx context.Context) Process {
	if v := ctx.Value(processKey{}); v != nil {
		return v.(Process)
	}

	return nil
}

// Clock gets the Clock used by the enclosing Process.
func Clock(ctx context.Context) clock.Clock {
	if proc := Self(ctx); proc != nil {
		return proc.Clock()
	}

	return nil
}

// Sleep pauses execution using the enclosing Process's clock. If the context is closed,
// Sleep returns false immediately.
func Sleep(ctx context.Context, d time.Duration) (uninterrupted bool) {
	clock := Clock(ctx)
	if clock == nil {
		if Self(ctx) == nil {
			panic("No enclosing Process found in context. geordi.Sleep() can only be called within a Process.")
		}
		panic("geordi.Process has no clock")
	}

	done := ctx.Done()
	timer := clock.Timer(d)

	select {
	case <-timer.C:
		return true
	case <-done:
		if !timer.Stop() {
			<-timer.C
		}
		return false
	}
}

func withProcess(ctx context.Context, proc Process) context.Context {
	return context.WithValue(ctx, processKey{}, proc)
}
