// Package task manage task, such as file upload, file copy between storages, offline download, etc.
package task

import (
	"context"
	"math"
	"runtime"
	"sync/atomic"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	PENDING   = "pending"
	RUNNING   = "running"
	SUCCEEDED = "succeeded"
	CANCELING = "canceling"
	CANCELED  = "canceled"
	ERRORED   = "errored"
)

type Func[K comparable] func(task *Task[K]) error
type Callback[K comparable] func(task *Task[K])

// Task is one unit of work. Its observable fields are written by the worker
// goroutine and read by anything holding a reference - the task list API, the
// upload progress endpoint, the manager's cleanup pass - so they have to be
// atomic. They used to be plain fields, which is a data race: a reader could
// observe a half-written string header (in Go a string is two words) and print
// garbage, and the race detector flagged every run.
type Task[K comparable] struct {
	ID   K
	Name string

	state    atomic.Pointer[string] // pending, running, canceling, canceled, succeeded, errored
	status   atomic.Pointer[string]
	progress atomic.Uint64 // math.Float64bits
	taskErr  atomic.Pointer[taskError]

	Func     Func[K]
	callback Callback[K]

	Ctx    context.Context
	cancel context.CancelFunc
}

func (t *Task[K]) SetStatus(status string) {
	t.status.Store(&status)
}

func (t *Task[K]) SetProgress(percentage float64) {
	t.progress.Store(math.Float64bits(percentage))
}

func (t *Task[K]) GetProgress() float64 {
	return math.Float64frombits(t.progress.Load())
}

func (t *Task[K]) GetState() string {
	if p := t.state.Load(); p != nil {
		return *p
	}
	return ""
}

func (t *Task[K]) GetStatus() string {
	if p := t.status.Load(); p != nil {
		return *p
	}
	return ""
}

// setState publishes a new state. It is the only write path, so the atomicity
// contract lives in one place.
func (t *Task[K]) setState(state string) {
	t.state.Store(&state)
}

// isState reports whether the current state equals any of the candidates.
func (t *Task[K]) isState(candidates ...string) bool {
	current := t.GetState()
	for _, c := range candidates {
		if current == c {
			return true
		}
	}
	return false
}

// taskError is a box so the atomic pointer can hold an error, which is an
// interface and therefore not comparable for storage in an atomic.Value.
type taskError struct{ err error }

// GetErr returns the task's error, if any.
func (t *Task[K]) GetErr() error {
	if p := t.taskErr.Load(); p != nil {
		return p.err
	}
	return nil
}

// setErr publishes the task's error.
func (t *Task[K]) setErr(err error) {
	if err == nil {
		t.taskErr.Store(nil)
		return
	}
	t.taskErr.Store(&taskError{err: err})
}

func (t *Task[K]) GetErrMsg() string {
	if err := t.GetErr(); err != nil {
		return err.Error()
	}
	return ""
}

func getCurrentGoroutineStack() string {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (t *Task[K]) run() {
	t.setState(RUNNING)
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("error [%s] while run task [%s],stack trace:\n%s", err, t.Name, getCurrentGoroutineStack())
			t.setErr(errors.Errorf("panic: %+v", err))
			t.setState(ERRORED)
		}
	}()
	t.setErr(t.Func(t))
	if t.GetErr() != nil {
		log.Errorf("error [%+v] while run task [%s]", t.GetErr(), t.Name)
	}
	if errors.Is(t.Ctx.Err(), context.Canceled) {
		t.setState(CANCELED)
	} else if t.GetErr() != nil {
		t.setState(ERRORED)
	} else {
		t.setState(SUCCEEDED)
		t.SetProgress(100)
		if t.callback != nil {
			t.callback(t)
		}
	}
}

func (t *Task[K]) retry() {
	t.run()
}

func (t *Task[K]) Done() bool {
	return t.isState(SUCCEEDED, CANCELED, ERRORED)
}

func (t *Task[K]) Cancel() {
	if t.isState(SUCCEEDED, CANCELED) {
		return
	}
	if t.cancel != nil {
		t.cancel()
	}
	// maybe can't cancel
	t.setState(CANCELING)
}

func WithCancelCtx[K comparable](task *Task[K]) *Task[K] {
	ctx, cancel := context.WithCancel(context.Background())
	task.Ctx = ctx
	task.cancel = cancel
	task.setState(PENDING)
	return task
}
