package api

import (
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/stat"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
)

var (
	logger = logging.GetDefaultLogger()
)

type TraceErrorOptions struct {
	count uint64
}

type TraceErrorOption func(*TraceErrorOptions)

// WithCount sets the error count.
func WithCount(count uint64) TraceErrorOption {
	return func(opts *TraceErrorOptions) {
		opts.count = count
	}
}

// TraceError records the provided error to the statistic structure of the target resource.
func TraceError(resource string, err error, opts ...TraceErrorOption) {
	if util.IsBlank(resource) || err == nil {
		return
	}
	defer func() {
		if e := recover(); e != nil {
			logger.Panicf("Fail to traceError, resource: %s, panic error: %+v", resource, e)
			return
		}
	}()

	if node := stat.GetResourceNode(resource); node != nil {
		var options = TraceErrorOptions{
			count: 1,
		}
		for _, opt := range opts {
			opt(&options)
		}
		traceErrorToNode(node, err, options.count)
	}
}

// TraceErrorToEntry records the provided error to the given SentinelEntry.
func TraceErrorToEntry(entry *base.SentinelEntry, err error, opts ...TraceErrorOption) {
	if entry == nil || err == nil {
		return
	}

	TraceErrorToCtx(entry.Context(), err, opts...)
}

// TraceErrorToCtx records the provided error to the given context.
func TraceErrorToCtx(ctx *base.EntryContext, err error, opts ...TraceErrorOption) {
	defer func() {
		if e := recover(); e != nil {
			logger.Panicf("Fail to traceErrorToCtx, parameter[ctx: %+v, err: %+v, opts: %+v], panic error: %+v", ctx, err, opts, e)
			return
		}
	}()

	if ctx == nil {
		return
	}
	node := ctx.StatNode
	if node == nil {
		logger.Warnf("Cannot trace error: nil StatNode in EntryContext, resource: %s", ctx.Resource.String())
		return
	}

	var options = TraceErrorOptions{
		count: 1,
	}
	for _, opt := range opts {
		opt(&options)
	}

	traceErrorToNode(node, err, options.count)
}

func traceErrorToNode(node base.StatNode, err error, cnt uint64) {
	if node == nil {
		return
	}
	if cnt <= 0 {
		return
	}
	node.AddMetric(base.MetricEventError, cnt)
}
