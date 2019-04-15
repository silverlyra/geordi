package geordi

import (
	"context"
	"fmt"
)

// Limit closes a Process after it has been restarted a certain number of times.
func Limit(maxRestarts uint) Response {
	return func(proc Process) Responder {
		return ResponderFn(func(ctx context.Context, err error) (Action, error) {
			if proc.Restarts() >= maxRestarts {
				return ActionClose, fmt.Errorf("Reached restart limit of %d", maxRestarts)
			}

			return ActionPass, err
		})
	}
}
