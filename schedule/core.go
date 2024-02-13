package schedule

import (
	"container/heap"
	"context"
	"github.com/fanck0605/http-schedule/config"
	"golang.org/x/sync/semaphore"
	"log"
	"sync"
)

// 结束标识
var sentinel *Task = nil

type Scheduler struct {
	// 调度器 pop 数据时，会借助这个 heap 来判断优先级
	heap TaskHeap
	// 用来传递数据的 queue，用于并发的接受数据
	queue chan *Task
	// 任务队列当前大小，其值为 len(heap) + len(queue)
	tasks chan struct{}
	// 循环一次为一个 tick
	tick chan struct{}
	// 控制任务开始和结束
	running sync.WaitGroup
}

func NewScheduler(maxTasks int) *Scheduler {
	return &Scheduler{
		heap:    make(TaskHeap, 0, maxTasks),
		queue:   make(chan *Task, maxTasks),
		tasks:   make(chan struct{}, maxTasks),
		tick:    make(chan struct{}),
		running: sync.WaitGroup{},
	}
}

// 通知继续循环
func (scheduler *Scheduler) notifyNext() {
	select {
	case scheduler.tick <- struct{}{}:
	default:
	}
}

// 等待继续循环
func (scheduler *Scheduler) waitNext() {
	<-scheduler.tick
}

// Push thread safe
func (scheduler *Scheduler) Push(task any) {
	scheduler.tasks <- struct{}{}
	scheduler.queue <- task.(*Task)
	scheduler.notifyNext()
}

// pop thread unsafe
func (scheduler *Scheduler) pop() any {
	<-scheduler.tasks
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

func (scheduler *Scheduler) Start() {
	scheduler.running.Add(1)
	go scheduler.run()
}

func (scheduler *Scheduler) Stop() {
	scheduler.Push(sentinel)
	scheduler.running.Wait()
}

// run 将会阻塞当前线程
func (scheduler *Scheduler) run() {
	defer scheduler.running.Done()

	resources := semaphore.NewWeighted(config.MaxResources)
	for {
		task := scheduler.pop().(*Task)
		if task == sentinel {
			log.Println("退出任务调度器！")
			break
		}
		ctx, cancel := context.WithCancel(context.Background())
		scheduled := make(chan struct{})
		go func() {
			cancelled := resources.Acquire(ctx, task.Resources)
			if cancelled != nil {
				scheduler.Push(task) // 如果被取消调度，送回队列重新调度
				close(scheduled)
				return
			}
			scheduler.notifyNext() // notify schedule next task
			close(scheduled)

			close(task.Ready)     // notify task ready to run
			<-task.Context.Done() // wait task running
			resources.Release(task.Resources)
		}()
		// 任务获取到资源，或者新来了任务，则继续
		scheduler.waitNext()
		cancel()
		<-scheduled
	}

	// waiting all task running!
	cancelled := resources.Acquire(context.Background(), config.MaxResources)
	if cancelled != nil {
		log.Printf("Error %s\n", cancelled)
	}
}
