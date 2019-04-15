package geordi

import (
	"context"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
)

// Process is a handle to a Task that was started.
type Process interface {
	Supervisor() Supervisor
	Task() Task
	Context() context.Context

	Cancel() Process
	Done() <-chan struct{}
	Join() (stopped bool, err error)

	Clock() clock.Clock

	Current() *Incarnation
	Previous() *Incarnation
	Restarts() uint
}

// Option configures a Process.
type Option func(*process)

func OnSuccess(responses ...Response) Option {
	return func(proc *process) { proc.onSuccess = makeResponder(proc, responses...) }
}
func OnError(responses ...Response) Option {
	return func(proc *process) { proc.onError = makeResponder(proc, responses...) }
}
func OnCancel(responses ...Response) Option {
	return func(proc *process) { proc.onCancel = makeResponder(proc, responses...) }
}
func OnPanic(responses ...Response) Option {
	return func(proc *process) { proc.onPanic = makeResponder(proc, responses...) }
}
func OnLinger(responses ...Response) Option {
	return func(proc *process) { proc.onLinger = makeResponder(proc, responses...) }
}
func Grace(d time.Duration) Option {
	return func(proc *process) { proc.grace = d }
}
func UseClock(c clock.Clock) Option {
	return func(proc *process) { proc.clock = c }
}

type process struct {
	sup    Supervisor
	task   Task
	ctx    context.Context
	cancel context.CancelFunc

	clock clock.Clock
	grace time.Duration

	onSuccess Responder
	onError   Responder
	onCancel  Responder
	onPanic   Responder
	onLinger  Responder

	life     chan struct{}
	current  *Incarnation
	previous *Incarnation
}

func spawn(ctx context.Context, sup Supervisor, task Task) *process {
	ctx, cancel := context.WithCancel(ctx)

	return &process{
		sup:    sup,
		task:   task,
		ctx:    ctx,
		cancel: cancel,
		life:   make(chan struct{}),
	}
}

func (p *process) Supervisor() Supervisor   { return p.sup }
func (p *process) Task() Task               { return p.task }
func (p *process) Context() context.Context { return p.ctx }

func (p *process) Cancel() Process {
	p.cancel()
	return p
}

func (p *process) Done() <-chan struct{} {
	return p.life
}

func (p *process) Join() (stopped bool, err error) {
	cur := p.current
	if cur == nil {
		return true, nil
	}

	select {
	case <-cur.taskLife:
		return true, cur.Error
	case <-cur.monitorLife:
		break
	}

	// If we got here, the monitor exited before we were told that the process task's goroutine
	// exited. That probably means the task didn't exit before its grace period expired, but it's
	// possible they both exited before our select was fulfilled, or that the task exited just
	// after the grace period expired.
	select {
	case <-cur.taskLife:
		return true, cur.Error
	default:
		return false, cur.Error
	}
}

func (p *process) Current() *Incarnation  { return p.current }
func (p *process) Previous() *Incarnation { return p.previous }

func (p *process) Restarts() uint {
	if cur := p.current; cur != nil {
		return cur.Index
	}
	return 0
}

func (p *process) Clock() clock.Clock { return p.clock }

func (p *process) run() {
	defer close(p.life)

	for {
		index := p.nextIndex()
		p.previous = p.current
		p.current = incarnate(p, index)

		err := p.current.runAndSupervise()

		action, err := p.responder(err).Respond(p.ctx, err)

		// TODO: check for context cancellation?

		switch action {
		case ActionRestart:
			continue
		case ActionClose:
			return
		case ActionCascade:
			if p.sup == nil {
				panic("the root geordi.Process has nothing to cascade to")
			}

			p.sup.Process().Cancel()
			return
		case ActionPass:
			panic("geordi.Process encountered ActionPass at the end of a response chain; no responders left to pass to")
		default:
			panic(fmt.Sprintf("geordi.Process encountered unknown Action value %v", action))
		}
	}
}

func (p *process) responder(err error) Responder {
	if err != nil {
		if err == context.Canceled {
			return p.onCancel
		} else if _, ok := err.(*LingerError); ok {
			return p.onLinger
		} else if _, ok := err.(*Panic); ok {
			return p.onPanic
		} else {
			return p.onError
		}
	}

	return p.onSuccess
}

func (p *process) nextIndex() uint {
	if cur := p.current; cur != nil {
		return cur.Index + 1
	}
	return 0
}
