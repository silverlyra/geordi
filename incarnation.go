package geordi

import (
	"time"

	"github.com/benbjohnson/clock"
)

// Incarnation is one execution of a Process.
type Incarnation struct {
	Index     uint
	StartedAt time.Time
	ExitedAt  time.Time
	Error     error

	proc        *process
	taskLife    chan struct{}
	monitorLife chan struct{}
}

func incarnate(proc *process, index uint) *Incarnation {
	return &Incarnation{
		Index:       index,
		proc:        proc,
		taskLife:    make(chan struct{}),
		monitorLife: make(chan struct{}),
	}
}

func (i *Incarnation) runAndSupervise() error {
	defer close(i.monitorLife)

	i.StartedAt = i.proc.clock.Now()
	canceled := i.proc.ctx.Done()

	var timeout *clock.Timer
	timedOut := make(chan struct{})
	defer func() {
		if timeout != nil {
			if !timeout.Stop() {
				<-timeout.C
			}
		}
	}()

	go i.run()

	select {
	case <-i.taskLife:
		return i.Error

	case <-canceled:
		if i.proc.grace > 0 {
			timeout = i.proc.clock.AfterFunc(i.proc.grace, func() { close(timedOut) })
		}
	}

	select {
	case <-i.taskLife:
		break

	case <-timedOut:
		i.Error = &LingerError{Grace: i.proc.grace}
	}

	return i.Error
}

func (i *Incarnation) run() {
	defer close(i.taskLife)

	defer func() {
		if p := recover(); p != nil {
			i.Error = &Panic{p}
		}

		i.ExitedAt = i.proc.clock.Now()
	}()

	err := i.proc.task(i.proc.ctx)
	if i.Error == nil {
		i.Error = err
	}
}
