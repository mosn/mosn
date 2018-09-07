// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxerrgroup implements an actor-runner with deterministic teardown. It is
// somewhat similar to package errgroup, except it does not require actor
// goroutines to understand context semantics. This makes it suitable for use in
// more circumstances; for example, goroutines which are handling connections
// from net.Listeners, or scanning input from a closable io.Reader.
//
// refers to github.com/oklog/run
package gxerrgroup

import (
	"github.com/AlexStocks/goext/runtime"
	"time"
)

// Group collects actors (functions) and runs them concurrently.
// When one actor (function) returns, all actors are interrupted.
// The zero value of a Group is useful.
type Group struct {
	actors []actor
	pool   *gxruntime.Pool
}

type actor struct {
	execute   func() error
	interrupt func(error)
}

// NewGroup initialize a Group instance which will create a goroutine pool.
// idleTimeout is a goroutine's max idle time interval.
func NewGroup(idleTimeout time.Duration) *Group {
	return &Group{
		actors: make([]actor, 0, 16),
		pool:   gxruntime.NewGoroutinePool(idleTimeout),
	}
}

func (g *Group) Close() {
	g.pool.Close()
}

// Add an actor (function) to the group. Each actor must be pre-emptable by an
// interrupt function. That is, if interrupt is invoked, execute should return.
// Also, it must be safe to call interrupt even after execute has returned.
//
// The first actor (function) to return interrupts all running actors.
// The error is passed to the interrupt functions, and is returned by Run.
func (g *Group) Add(execute func() error, interrupt func(error)) {
	g.actors = append(g.actors, actor{execute, interrupt})
}

// Run all actors (functions) concurrently.
// When the first actor returns, all others are interrupted.
// Run only returns when all actors have exited.
// Run returns the error returned by the first exiting actor.
func (g *Group) Run() error {
	if len(g.actors) == 0 {
		return nil
	}

	// Run each actor.
	errors := make(chan error, len(g.actors))
	for idx := range g.actors {
		i := idx
		a := g.actors[i]
		g.pool.Go(func() {
			errors <- a.execute()
		})
		//go func(a actor) {
		//	fmt.Printf("a:%+v\n", a)
		//	errors <- a.execute()
		//}(a)
	}

	// Wait for the first actor to stop.
	err := <-errors

	// Signal all actors to stop.
	for _, a := range g.actors {
		a.interrupt(err)
	}

	// Wait for all actors to stop.
	for i := 1; i < cap(errors); i++ {
		<-errors
	}

	// Return the original error.
	return err
}
