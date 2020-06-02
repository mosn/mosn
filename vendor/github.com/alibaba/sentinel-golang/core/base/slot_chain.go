package base

import (
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"sync"
)

var logger = logging.GetDefaultLogger()

// StatPrepareSlot is responsible for some preparation before statistic
// For example: init structure and so on
type StatPrepareSlot interface {
	// Prepare function do some initialization
	// Such as: init statistic structure、node and etc
	// The result of preparing would store in EntryContext
	// All StatPrepareSlots execute in sequence
	// Prepare function should not throw panic.
	Prepare(ctx *EntryContext)
}

// RuleCheckSlot is rule based checking strategy
// All checking rule must implement this interface.
type RuleCheckSlot interface {
	// Check function do some validation
	// It can break off the slot pipeline
	// Each TokenResult will return check result
	// The upper logic will control pipeline according to SlotResult.
	Check(ctx *EntryContext) *TokenResult
}

// StatSlot is responsible for counting all custom biz metrics.
// StatSlot would not handle any panic, and pass up all panic to slot chain
type StatSlot interface {
	// OnEntryPass function will be invoked when StatPrepareSlots and RuleCheckSlots execute pass
	// StatSlots will do some statistic logic, such as QPS、log、etc
	OnEntryPassed(ctx *EntryContext)
	// OnEntryBlocked function will be invoked when StatPrepareSlots and RuleCheckSlots fail to execute
	// It may be inbound flow control or outbound cir
	// StatSlots will do some statistic logic, such as QPS、log、etc
	// blockError introduce the block detail
	OnEntryBlocked(ctx *EntryContext, blockError *BlockError)
	// onComplete function will be invoked when chain exits.
	// The request may be executed successful or blocked or internal error
	OnCompleted(ctx *EntryContext)
}

// SlotChain hold all system slots and customized slot.
// SlotChain support plug-in slots developed by developer.
type SlotChain struct {
	statPres   []StatPrepareSlot
	ruleChecks []RuleCheckSlot
	stats      []StatSlot
	// EntryContext Pool, used for reuse EntryContext object
	pool sync.Pool
}

func NewSlotChain() *SlotChain {
	return &SlotChain{
		statPres:   make([]StatPrepareSlot, 0, 5),
		ruleChecks: make([]RuleCheckSlot, 0, 5),
		stats:      make([]StatSlot, 0, 5),
		pool: sync.Pool{
			New: func() interface{} {
				return NewEmptyEntryContext()
			},
		},
	}
}

// Get a EntryContext from EntryContext pool, if pool doesn't have enough EntryContext then new one.
func (sc *SlotChain) GetPooledContext() *EntryContext {
	ctx := sc.pool.Get().(*EntryContext)
	ctx.startTime = util.CurrentTimeMillis()
	return ctx
}

func (sc *SlotChain) RefurbishContext(c *EntryContext) {
	if c != nil {
		c.Reset()
		sc.pool.Put(c)
	}
}

func (sc *SlotChain) AddStatPrepareSlotFirst(s StatPrepareSlot) {
	ns := make([]StatPrepareSlot, 0, len(sc.statPres)+1)
	// add to first
	ns = append(ns, s)
	sc.statPres = append(ns, sc.statPres...)
}

func (sc *SlotChain) AddStatPrepareSlotLast(s StatPrepareSlot) {
	sc.statPres = append(sc.statPres, s)
}

func (sc *SlotChain) AddRuleCheckSlotFirst(s RuleCheckSlot) {
	ns := make([]RuleCheckSlot, 0, len(sc.ruleChecks)+1)
	ns = append(ns, s)
	sc.ruleChecks = append(ns, sc.ruleChecks...)
}

func (sc *SlotChain) AddRuleCheckSlotLast(s RuleCheckSlot) {
	sc.ruleChecks = append(sc.ruleChecks, s)
}

func (sc *SlotChain) AddStatSlotFirst(s StatSlot) {
	ns := make([]StatSlot, 0, len(sc.stats)+1)
	ns = append(ns, s)
	sc.stats = append(ns, sc.stats...)
}

func (sc *SlotChain) AddStatSlotLast(s StatSlot) {
	sc.stats = append(sc.stats, s)
}

// The entrance of slot chain
// Return the TokenResult and nil if internal panic.
func (sc *SlotChain) Entry(ctx *EntryContext) *TokenResult {
	// This should not happen, unless there are errors existing in Sentinel internal.
	// If happened, need to add TokenResult in EntryContext
	defer func() {
		if err := recover(); err != nil {
			logger.Panicf("Sentinel internal panic in SlotChain, err: %+v", err)
			return
		}
	}()

	// execute prepare slot
	sps := sc.statPres
	if len(sps) > 0 {
		for _, s := range sps {
			s.Prepare(ctx)
		}
	}

	// execute rule based checking slot
	rcs := sc.ruleChecks
	ruleCheckRet := NewTokenResultPass()
	if len(rcs) > 0 {
		for _, s := range rcs {
			sr := s.Check(ctx)
			ctx.Output.LastResult = sr
			// check slot result
			if sr.status == ResultStatusBlocked {
				ruleCheckRet = sr
				break
			}

			// This slot passed, continue.
		}
	}
	ctx.Output.LastResult = ruleCheckRet

	// execute statistic slot
	ss := sc.stats
	if len(ss) > 0 {
		for _, s := range ss {
			// indicate the result of rule based checking slot.
			if !ruleCheckRet.IsBlocked() {
				s.OnEntryPassed(ctx)
			} else {
				// The block error should not be nil.
				s.OnEntryBlocked(ctx, ruleCheckRet.blockErr)
			}
		}
	}
	return ruleCheckRet
}

func (sc *SlotChain) exit(ctx *EntryContext) {
	for _, s := range sc.stats {
		s.OnCompleted(ctx)
	}
	// relieve the context here
}
