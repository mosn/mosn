package base

type EntryContext struct {
	// Use to calculate RT
	startTime uint64

	Resource *ResourceWrapper
	StatNode StatNode

	Input  *SentinelInput
	Output *SentinelOutput
}

func (ctx *EntryContext) StartTime() uint64 {
	return ctx.startTime
}

func (ctx *EntryContext) IsBlocked() bool {
	if ctx.Output.LastResult == nil {
		return false
	}
	return ctx.Output.LastResult.IsBlocked()
}

func NewEmptyEntryContext() *EntryContext {
	ctx := &EntryContext{
		Input:  nil,
		Output: newEmptyOutput(),
	}
	return ctx
}

type SentinelInput struct {
	AcquireCount uint32
	Flag         int32
	Args         []interface{}

	// store some values in this context when calling context in slot.
	Attachments map[interface{}]interface{}
}

func newEmptyInput() *SentinelInput {
	return &SentinelInput{}
}

type SentinelOutput struct {
	LastResult *TokenResult

	// store output data.
	Attachments map[interface{}]interface{}
}

func newEmptyOutput() *SentinelOutput {
	return &SentinelOutput{}
}

// Reset init EntryContext,
func (ctx *EntryContext) Reset() {
	// reset all fields of ctx
	ctx.startTime = 0
	ctx.Resource = nil
	ctx.StatNode = nil
	ctx.Input = nil
	ctx.Output = newEmptyOutput()
}
