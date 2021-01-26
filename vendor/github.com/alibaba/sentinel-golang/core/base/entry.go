package base

import (
	"sync"
)

type SentinelEntry struct {
	res *ResourceWrapper
	// one entry with one context
	ctx *EntryContext
	// each entry holds a slot chain.
	// it means this entry will go through the sc
	sc *SlotChain

	exitCtl sync.Once
}

func (e *SentinelEntry) Context() *EntryContext {
	return e.ctx
}

func (e *SentinelEntry) Resource() *ResourceWrapper {
	return e.res
}

func NewSentinelEntry(ctx *EntryContext, rw *ResourceWrapper, sc *SlotChain) *SentinelEntry {
	return &SentinelEntry{res: rw, ctx: ctx, sc: sc}
}

func (e *SentinelEntry) Exit() {
	ctx := e.ctx
	e.exitCtl.Do(func() {
		if e.sc != nil {
			e.sc.exit(ctx)
			e.sc.RefurbishContext(ctx)
		}
	})
}
