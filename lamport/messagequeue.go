package main

import (
    "container/heap"
    "fmt"
    "sort"
)

type MessageQ []LamportMessage

func (h MessageQ) Len() int           { return len(h) }
func (h MessageQ) Less(i, j int) bool { return h[i].TS < h[j].TS }
func (h MessageQ) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *MessageQ) Push(x interface{}) {
    *h = append(*h, x.(LamportMessage))
}

func (h *MessageQ) Pop() interface{} {
    old := *h
    n := len(old)
    x := old[n-1]
    *h = old[0 : n-1]
    return x
}

func MessageQTest() {
    h := &MessageQ{}
    heap.Init(h)
    heap.Push(h, *NewRequest(1, 1, 2))
    heap.Push(h, *NewRequest(0, 2, 1))
    heap.Push(h, *NewRequest(2, 1, 3))
    heap.Push(h, *NewRequest(10, 3, 2))
    fmt.Println(*h)
    sort.Sort(MessageQ(*h))
    fmt.Println((*h)[0].String())
    //heap.Pop(h)
    fmt.Println(*h)
}

