package geordi

import "context"

// Task is a workload to be run within a Supervisor.
type Task func(ctx context.Context) error
