// Copyright 2020 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build proxytest

package proxywasm

func VMStateReset() {
	// (@mathetake) I assume that the currentState be protected by lock on hostMux
	currentState = &state{
		rootContexts:      make(map[uint32]*rootContextState),
		httpStreams:       make(map[uint32]HttpContext),
		streams:           make(map[uint32]StreamContext),
		contextIDToRootID: make(map[uint32]uint32),
	}
}

func VMStateGetActiveContextID() uint32 {
	return currentState.activeContextID
}
