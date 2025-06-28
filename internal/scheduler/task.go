package scheduler

import (
	"container/heap"
	"time"

	"payer-status-io/internal/config"
)

// Task represents a scheduled probe task
type Task struct {
	Payer    string
	Endpoint config.Endpoint
	NextRun  time.Time
	Interval time.Duration
}

// TaskHeap implements heap.Interface for Task scheduling
type TaskHeap []*Task

func (h TaskHeap) Len() int { return len(h) }

func (h TaskHeap) Less(i, j int) bool {
	return h[i].NextRun.Before(h[j].NextRun)
}

func (h TaskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *TaskHeap) Push(x interface{}) {
	*h = append(*h, x.(*Task))
}

func (h *TaskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	task := old[n-1]
	*h = old[0 : n-1]
	return task
}

// Peek returns the next task without removing it
func (h TaskHeap) Peek() *Task {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}

// NewTaskHeap creates a new task heap
func NewTaskHeap() *TaskHeap {
	h := &TaskHeap{}
	heap.Init(h)
	return h
}

// PushTask adds a task to the heap
func (h *TaskHeap) PushTask(task *Task) {
	heap.Push(h, task)
}

// PopTask removes and returns the next task
func (h *TaskHeap) PopTask() *Task {
	if h.Len() == 0 {
		return nil
	}
	return heap.Pop(h).(*Task)
}

// PeekTask returns the next task without removing it
func (h *TaskHeap) PeekTask() *Task {
	return h.Peek()
}
