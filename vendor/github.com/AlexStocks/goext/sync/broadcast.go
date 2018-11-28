// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// ref: https://rogpeppe.wordpress.com/2009/12/01/concurrent-idioms-1-broadcasting-values-in-go-with-linked-channels/
package gxsync

import (
	"fmt"
)

var (
	ErrBroadcastClosed = fmt.Errorf("broadcast closed!")
)

type Node struct {
	next  chan Node
	value interface{}
}

type Broadcaster struct {
	registry chan chan (chan Node)
	data     chan<- interface{}
}

type Receiver struct {
	C chan Node
}

// create a new broadcaster object.
func NewBroadcaster() Broadcaster {
	registry := make(chan (chan (chan Node)))
	data := make(chan interface{})
	go func() {
		var (
			value  interface{}
			cursor chan Node
			node   chan Node
			reader chan chan Node
		)
		cursor = make(chan Node, 1)
		for {
			select {
			case value = <-data:
				if value == nil {
					cursor <- Node{}
					return
				}
				node = make(chan Node, 1)
				cursor <- Node{next: node, value: value}
				cursor = node
			case reader = <-registry:
				reader <- cursor
			}
		}
	}()
	return Broadcaster{
		registry: registry,
		data:     data,
	}
}

// start listening to the nodes.
func (b Broadcaster) Listen() Receiver {
	r := make(chan chan Node, 0)
	b.registry <- r
	return Receiver{<-r}
}

// Node a value to all listeners.
func (b Broadcaster) Write(value interface{}) (rerr error) {
	defer func() {
		if e := recover(); e != nil {
			rerr = fmt.Errorf("panic error:%value", e)
		}
	}()

	if value == nil {
		close(b.data)
		return
	}
	b.data <- value

	return
}

func (b Broadcaster) Close() { b.Write(nil) }

// read a value that has been broadcast,
// waiting until one is available if necessary.
func (r *Receiver) Read() (interface{}, error) {
	n := <-r.C
	value := n.value
	r.C <- n
	r.C = n.next

	if value == nil {
		return nil, ErrBroadcastClosed
	}

	return value, nil
}

func (r *Receiver) ReadChan() chan Node {
	return r.C
}

func (r *Receiver) ReadDone(n Node) (interface{}, error) {
	r.C <- n
	r.C = n.next

	if n.value == nil {
		return nil, ErrBroadcastClosed
	}

	return n.value, nil
}
