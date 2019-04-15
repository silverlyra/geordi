package geordi

import (
	"context"
)

// Response returns a desired responder for a process.
type Response func(proc Process) Responder

func Close() Response {
	return func(proc Process) Responder {
		return ResponderFn(func(ctx context.Context, err error) (Action, error) {
			return ActionClose, err
		})
	}
}
func Restart() Response {
	return func(proc Process) Responder {
		return ResponderFn(func(ctx context.Context, err error) (Action, error) {
			return ActionRestart, err
		})
	}
}
func Cascade() Response {
	return func(proc Process) Responder {
		return ResponderFn(func(ctx context.Context, err error) (Action, error) {
			return ActionCascade, err
		})
	}
}

// Responder handles a process event.
type Responder interface {
	Respond(ctx context.Context, err error) (Action, error)
}

// ResponderFn makes a plain function usable as a Responder.
type ResponderFn func(ctx context.Context, err error) (Action, error)

// Respond implements Responder for ResponderFn.
func (r ResponderFn) Respond(ctx context.Context, err error) (Action, error) {
	return r(ctx, err)
}

// Responders is a series of responders that will be tried in order.
type Responders []Responder

// Respond makes a slice of Responders implement Responder.
func (rs Responders) Respond(ctx context.Context, err error) (Action, error) {
	for _, r := range rs {
		if action, err := r.Respond(ctx, err); action != ActionPass {
			return action, err
		}
	}

	return ActionPass, err
}

func makeResponder(proc Process, responses ...Response) Responder {
	responders := make(Responders, 0, len(responses))
	for _, response := range responses {
		responders = append(responders, response(proc))
	}

	return responders
}

// Action is a response to take when a Task returns or panics.
type Action int

const (
	ActionPass Action = iota
	ActionRestart
	ActionClose
	ActionCascade
)

// When applies a response conditionally. It can be used (for example) to retry a task only when
// a retryable error occurs.
func When(cond func(ctx context.Context, err error) bool, responses ...Response) Response {
	return func(proc Process) Responder {
		return &when{cond, makeResponder(proc, responses...)}
	}
}

type when struct {
	cond func(ctx context.Context, err error) bool
	resp Responder
}

func (w when) Respond(ctx context.Context, err error) (Action, error) {
	if w.cond(ctx, err) {
		return w.resp.Respond(ctx, err)
	}

	return ActionPass, err
}
