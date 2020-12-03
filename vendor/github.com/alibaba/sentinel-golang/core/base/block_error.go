package base

// BlockError indicates the request was blocked by Sentinel.
type BlockError struct {
	blockType BlockType
	blockMsg  string

	rule          SentinelRule
	snapshotValue interface{}
}

func (e *BlockError) BlockMsg() string {
	return e.blockMsg
}

func (e *BlockError) BlockType() BlockType {
	return e.blockType
}

func (e *BlockError) TriggeredRule() SentinelRule {
	return e.rule
}

func (e *BlockError) TriggeredValue() interface{} {
	return e.snapshotValue
}

func NewBlockError(blockType BlockType, blockMsg string) *BlockError {
	return &BlockError{blockType: blockType, blockMsg: blockMsg}
}

func NewBlockErrorWithCause(blockType BlockType, blockMsg string, rule SentinelRule, snapshot interface{}) *BlockError {
	return &BlockError{blockType: blockType, blockMsg: blockMsg, rule: rule, snapshotValue: snapshot}
}

func (e *BlockError) Error() string {
	return "SentinelBlockException: " + e.blockMsg
}
