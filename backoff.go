package geordi

import (
	"context"
	"fmt"

	"github.com/cenkalti/backoff"
)

// Backoff applies an exponential backoff to restarting a Process.
func Backoff(options ...func(bo *backoff.ExponentialBackOff)) Response {
	return func(proc Process) Responder {
		bo := backoff.NewExponentialBackOff()
		bo.Clock = proc.Clock()

		for _, opt := range options {
			opt(bo)
		}

		return &backoffResponder{bo, proc}
	}
}

type backoffResponder struct {
	*backoff.ExponentialBackOff
	proc Process
}

func (b backoffResponder) Respond(ctx context.Context, err error) (Action, error) {
	wait := b.NextBackOff()
	if wait == backoff.Stop {
		return ActionClose, fmt.Errorf("Backoff reached time limit: %s", b.MaxElapsedTime)
	}

	if !Sleep(ctx, wait) {
		return ActionClose, fmt.Errorf("Context closed while sleeping for backoff")
	}

	return ActionRestart, nil
}
