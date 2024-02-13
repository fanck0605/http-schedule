package schedule

import (
	"context"
)

type Task struct {
	// 调度优先级
	Priority int64
	// 调度需要的资源数
	Resources int64
	// 如果有资源了，通知 Gin 继续执行
	Ready chan<- struct{}
	// Gin 任务上下文，判定任务是否 Done
	Context context.Context
}

type TaskHeap []*Task

func (heap TaskHeap) Len() int {
	return len(heap)
}

// Less Priority 越大优先级越高
func (heap TaskHeap) Less(l, r int) bool {
	lv := heap[l]
	rv := heap[r]
	if lv == nil {
		return false
	} else if rv == nil {
		return true
	} else {
		return lv.Priority > rv.Priority
	}
}

func (heap TaskHeap) Swap(l, r int) {
	heap[l], heap[r] = heap[r], heap[l]
}

func (heap *TaskHeap) Push(v any) {
	*heap = append(*heap, v.(*Task))
}

func (heap *TaskHeap) Pop() any {
	old := *heap
	length := len(old)
	popped := old[length-1]
	old[length-1] = nil // avoid memory leak
	*heap = old[:length-1]
	return popped
}
