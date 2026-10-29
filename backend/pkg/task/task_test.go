package task

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
)

func TestTask_Manager(t *testing.T) {
	tm := NewTaskManager(3, func(id *uint64) {
		atomic.AddUint64(id, 1)
	})
	id := tm.Submit(WithCancelCtx(&Task[uint64]{
		Name: "test",
		Func: func(task *Task[uint64]) error {
			time.Sleep(time.Millisecond * 500)
			return nil
		},
	}))
	task, ok := tm.Get(id)
	if !ok {
		t.Fatal("task not found")
	}
	time.Sleep(time.Millisecond * 100)
	if task.GetState() != RUNNING {
		t.Errorf("task status not running: %s", task.GetState())
	}
	time.Sleep(time.Second)
	if task.GetState() != SUCCEEDED {
		t.Errorf("task status not finished: %s", task.GetState())
	}
}

func TestTask_Cancel(t *testing.T) {
	tm := NewTaskManager(3, func(id *uint64) {
		atomic.AddUint64(id, 1)
	})
	id := tm.Submit(WithCancelCtx(&Task[uint64]{
		Name: "test",
		Func: func(task *Task[uint64]) error {
			for {
				if utils.IsCanceled(task.Ctx) {
					return nil
				} else {
					t.Logf("task is running")
				}
			}
		},
	}))
	task, ok := tm.Get(id)
	if !ok {
		t.Fatal("task not found")
	}
	time.Sleep(time.Microsecond * 50)
	task.Cancel()
	time.Sleep(time.Millisecond)
	if task.GetState() != CANCELED {
		t.Errorf("task status not canceled: %s", task.GetState())
	}
}

func TestTask_Retry(t *testing.T) {
	tm := NewTaskManager(3, func(id *uint64) {
		atomic.AddUint64(id, 1)
	})
	num := 0
	id := tm.Submit(WithCancelCtx(&Task[uint64]{
		Name: "test",
		Func: func(task *Task[uint64]) error {
			num++
			if num&1 == 1 {
				return errors.New("test error")
			}
			return nil
		},
	}))
	task, ok := tm.Get(id)
	if !ok {
		t.Fatal("task not found")
	}
	time.Sleep(time.Millisecond)
	if task.GetErr() == nil {
		t.Error(task.GetState())
		t.Fatal("task error is nil, but expected error")
	} else {
		t.Logf("task error: %s", task.GetErr())
	}
	task.retry()
	time.Sleep(time.Millisecond)
	if task.GetErr() != nil {
		t.Errorf("task error: %+v, but expected nil", task.GetErr())
	}
}
