// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base

import "fmt"

// BlockError indicates the request was blocked by Sentinel.
type BlockError struct {
	blockType BlockType
	// blockMsg provides additional message for the block error.
	blockMsg string

	rule SentinelRule
	// snapshotValue represents the triggered "snapshot" value
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

func NewBlockErrorFromDeepCopy(from *BlockError) *BlockError {
	return &BlockError{
		blockType:     from.blockType,
		blockMsg:      from.blockMsg,
		rule:          from.rule,
		snapshotValue: from.snapshotValue,
	}
}

func NewBlockError(blockType BlockType) *BlockError {
	return &BlockError{blockType: blockType}
}

func NewBlockErrorWithMessage(blockType BlockType, message string) *BlockError {
	return &BlockError{blockType: blockType, blockMsg: message}
}

func NewBlockErrorWithCause(blockType BlockType, blockMsg string, rule SentinelRule, snapshot interface{}) *BlockError {
	return &BlockError{blockType: blockType, blockMsg: blockMsg, rule: rule, snapshotValue: snapshot}
}

func (e *BlockError) Error() string {
	if len(e.blockMsg) == 0 {
		return fmt.Sprintf("SentinelBlockError: %s", e.blockType.String())
	}
	return fmt.Sprintf("SentinelBlockError: %s, message: %s", e.blockType.String(), e.blockMsg)
}
