package stat

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

type StatNodePrepareSlot struct {
}

func (s *StatNodePrepareSlot) Prepare(ctx *base.EntryContext) {
	node := GetOrCreateResourceNode(ctx.Resource.Name(), ctx.Resource.Classification())
	// Set the resource node to the context.
	ctx.StatNode = node
}
