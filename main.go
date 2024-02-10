package main

import (
	"container/heap"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/semaphore"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type PriorityQueue struct {
	// 调度器空闲时，会借助这个 heap 来判断优先级
	heap ContextHeap
	// 用来传递数据的 queue，用于并发的接受数据
	queue chan *Context
	// 优先队列当前大小，其值为 len(heap) + len(queue)
	semaphore chan struct{}
}

func NewPriorityQueue(maxSize int) PriorityQueue {
	return PriorityQueue{
		heap:      make(ContextHeap, 0, maxSize),
		queue:     make(chan *Context, maxSize),
		semaphore: make(chan struct{}, maxSize),
	}
}

// Push thread safe
func (que *PriorityQueue) Push(item any) {
	que.queue <- item.(*Context)
	que.semaphore <- struct{}{}
}

// Pop thread unsafe
func (que *PriorityQueue) Pop() any {
	<-que.semaphore
LOOP:
	for {
		select {
		case item := <-que.queue:
			heap.Push(&que.heap, item)
		default:
			break LOOP
		}
	}
	return heap.Pop(&que.heap)
}

func requestMonitor(wg *sync.WaitGroup, que PriorityQueue) {
	ctx := context.Background()
	sem := semaphore.NewWeighted(maxResources)
LOOP:
	for {
		item := que.Pop().(*Context)
		if item == nil {
			log.Println("退出请求监控器！")
			break LOOP
		}
		if err := sem.Acquire(ctx, item.Weight); err != nil {
			log.Fatal("error")
		}
		log.Println("schedule task " + item.Request.RequestURI)
		go func() {
			defer sem.Release(item.Weight)
			time.Sleep(time.Duration(item.Priority) * time.Second)
			log.Println("Context: " + item.Request.RequestURI)
			item.Result <- item.Request.RequestURI
		}()
		select {
		case <-ctx.Done(): // 等待上级通知
			log.Println("退出请求监控器！")
			break LOOP
		default:
		}
	}

	if err := sem.Acquire(ctx, maxResources); err != nil {
		log.Fatal("error")
	}
	wg.Done()
}

func main() {
	requestQueue := NewPriorityQueue(10)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go requestMonitor(&wg, requestQueue)

	router := gin.Default()
	router.Any("*uri", func(ctx *gin.Context) {
		future := make(chan string, 0)
		requestQueue.Push(&Context{
			Priority: getPriority(ctx.Request),
			Weight:   getWeight(ctx.Request),
			Request:  ctx.Request,
			Result:   future,
		})
		ctx.JSON(http.StatusOK, gin.H{
			"message": <-future,
		})
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	// The Context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}
	var semimetal *Context
	requestQueue.Push(semimetal)
	wg.Wait()
	log.Println("Server exiting")
}
