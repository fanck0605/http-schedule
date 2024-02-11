package schedule

import (
	"container/heap"
	"context"
	"github.com/fanck0605/http-schedule/config"
	"golang.org/x/sync/semaphore"
	"log"
	"sync"
)

// chan struct{} 专用占位符
var empty = struct{}{}

// 结束标识
var sentinel *Task = nil

type Scheduler struct {
	// 调度器 unsafePop 数据时，会借助这个 heap 来判断优先级
	heap TaskHeap
	// 用来传递数据的 queue，用于并发的接受数据
	queue chan *Task
	// 优先队列当前大小，其值为 len(heap) + len(queue)
	semaphore chan struct{}
	// 控制任务开始和结束
	waiter sync.WaitGroup
}

func NewScheduler(maxTasks int) Scheduler {
	return Scheduler{
		heap:      make(TaskHeap, 0, maxTasks),
		queue:     make(chan *Task, maxTasks),
		semaphore: make(chan struct{}, maxTasks),
		waiter:    sync.WaitGroup{},
	}
}

// Push thread safe
func (scheduler *Scheduler) Push(task any) {
	scheduler.queue <- task.(*Task)
	scheduler.semaphore <- empty
}

// unsafePop thread unsafe
func (scheduler *Scheduler) unsafePop() any {
	<-scheduler.semaphore
LOOP:
	for {
		select {
		case task := <-scheduler.queue:
			heap.Push(&scheduler.heap, task)
		default:
			break LOOP
		}
	}
	return heap.Pop(&scheduler.heap)
}

// run 将会阻塞当前线程
func (scheduler *Scheduler) run() {
	bg := context.Background()
	sem := semaphore.NewWeighted(config.MaxWeight)

	for {
		task := scheduler.unsafePop().(*Task)
		if task == sentinel {
			log.Println("退出任务调度器！")
			break
		}
		if err := sem.Acquire(bg, task.Weight); err != nil {
			log.Println(err)
		} else {
			go func() {
				log.Printf("Schedule task %p", task)
				task.Ready <- empty
				log.Printf("Waiting task waiter %p", task)
				<-task.Context.Done()
				sem.Release(task.Weight)
				log.Printf("Context waiter %p", task)
			}()
		}
	}
	// waiting all task waiter!
	if err := sem.Acquire(bg, config.MaxWeight); err != nil {
		log.Printf("Error %s\n", err)
	}
	scheduler.waiter.Done()
}

func (scheduler *Scheduler) Start() {
	scheduler.waiter.Add(1)
	go scheduler.run()
}

func (scheduler *Scheduler) Stop() {
	scheduler.Push(sentinel)
	scheduler.waiter.Wait()
}
