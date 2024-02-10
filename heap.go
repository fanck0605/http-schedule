package main

import (
	"net/http"
)

type Context struct {
	Priority int64
	Weight   int64
	Request  *http.Request
	Result   chan<- string
}

type ContextHeap []*Context

func (que ContextHeap) Len() int {
	return len(que)
}

func (que ContextHeap) Less(i, j int) bool {
	return que[i].Priority > que[j].Priority
}

func (que ContextHeap) Swap(i, j int) {
	que[i], que[j] = que[j], que[i]
}

func (que *ContextHeap) Push(item any) {
	*que = append(*que, item.(*Context))
}

func (que *ContextHeap) Pop() any {
	length := len(*que)
	popped := (*que)[length-1]
	*que = (*que)[:length-1]
	return popped
}
