package geordi

import (
	"context"
	"time"

	"github.com/benbjohnson/clock"
)

// Supervisor runs tasks in isolated goroutines, and can respond reliably when they finish or fail.
type Supervisor interface {
	Start(ctx context.Context, task Task, options ...Option) Process
	Process() Process
}

type supervisor struct {
	self Process
	ctx  context.Context

	starts    chan *process
	exits     chan *process
	processes map[*process]struct{}

	defaults []Option
}

// New returns a Task that runs a Supervisor.
func New(options ...func(Supervisor)) Task {
	return func(ctx context.Context) error {
		sup := &supervisor{
			self:      Self(ctx),
			ctx:       ctx,
			starts:    make(chan *process, 64),
			exits:     make(chan *process),
			processes: make(map[*process]struct{}),
			defaults:  append([]Option{}, DefaultDefaults...),
		}

		for _, o := range options {
			o(sup)
		}

		return sup.run()
	}
}

// Start spawns a goroutine and runs the given task.
func (s *supervisor) Start(ctx context.Context, task Task, options ...Option) Process {
	proc := spawn(ctx, s, task)

	for _, o := range s.defaults {
		o(proc)
	}
	for _, o := range options {
		o(proc)
	}

	s.starts <- proc
	return proc
}

func (s *supervisor) Process() Process {
	return s.self
}

func (s *supervisor) run() error {
	s.supervise()
	s.shutdown()

	return nil
}

func (s *supervisor) supervise() {
	done := s.ctx.Done()

	for {
		select {
		case proc := <-s.starts:
			s.processes[proc] = struct{}{}
			proc.run()
		case proc := <-s.exits:
			delete(s.processes, proc)
		case <-done:
			return
		}
	}
}

func (s *supervisor) shutdown() {
	for proc := range s.processes {
		proc.Cancel()
	}

	for len(s.processes) > 0 {
		select {
		case <-s.starts:
			// ignore
			continue
		case proc := <-s.exits:
			delete(s.processes, proc)
		}
	}
}

// Defaults sets default options for Processes run under a Supervisor.
func Defaults(options ...Option) func(Supervisor) {
	return func(sup Supervisor) {
		if s, ok := sup.(*supervisor); ok {
			s.defaults = append(s.defaults, options...)
		}

		panic("geordi.Defaults can only be applied to a *geordi.supervisor")
	}
}

// DefaultDefaults are the default options added to new Supervisors.
var DefaultDefaults = []Option{
	OnSuccess(Close()),
	OnCancel(Close()),
	OnError(Close()),
	OnPanic(Close()),
	OnLinger(Cascade()),
	Grace(2 * time.Minute),
	UseClock(clock.New()),
}
