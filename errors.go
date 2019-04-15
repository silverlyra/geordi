package geordi

import (
	"fmt"
	"time"
)

// A Panic is a wrapper around a Task's panic.
type Panic struct {
	Value interface{}
}

func (p Panic) Error() string {
	return fmt.Sprintf("panic: %v", p.Value)
}

// LingerError means a task context was cancelled, but did not return within
// its process's grace period.
type LingerError struct {
	Grace time.Duration
}

func (e LingerError) Error() string {
	return fmt.Sprintf("Process context was cancelled, but task did not return within grace period %s", e.Grace)
}
