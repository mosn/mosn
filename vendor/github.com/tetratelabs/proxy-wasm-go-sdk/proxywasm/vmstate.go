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

package proxywasm

import (
	"errors"
	"fmt"
)

type (
	HttpCalloutCallBack = func(numHeaders, bodySize, numTrailers int)

	rootContextState struct {
		context       RootContext
		httpCallbacks map[uint32]*struct {
			callback        HttpCalloutCallBack
			callerContextID uint32
		}
	}
)

type state struct {
	newRootContext   func(contextID uint32) RootContext
	rootContexts     map[uint32]*rootContextState
	newStreamContext func(rootContextID, contextID uint32) StreamContext
	streams          map[uint32]StreamContext
	newHttpContext   func(rootContextID, contextID uint32) HttpContext
	httpStreams      map[uint32]HttpContext

	contextIDToRootID map[uint32]uint32
	activeContextID   uint32
}

var currentState = &state{
	rootContexts:      make(map[uint32]*rootContextState),
	httpStreams:       make(map[uint32]HttpContext),
	streams:           make(map[uint32]StreamContext),
	contextIDToRootID: make(map[uint32]uint32),
}

func SetNewRootContext(f func(contextID uint32) RootContext) {
	currentState.newRootContext = f
}

func SetNewHttpContext(f func(rootContextID, contextID uint32) HttpContext) {
	currentState.newHttpContext = f
}

func SetNewStreamContext(f func(rootContextID, contextID uint32) StreamContext) {
	currentState.newStreamContext = f
}

var ErrorRootContextNotFound = errors.New("root context not found")

func GetRootContextByID(rootContextID uint32) (RootContext, error) {
	rootContextState, ok := currentState.rootContexts[rootContextID]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrorRootContextNotFound, rootContextID)
	}
	return rootContextState.context, nil
}

//go:inline
func (s *state) createRootContext(contextID uint32) {
	var ctx RootContext
	if s.newRootContext == nil {
		ctx = &DefaultRootContext{}
	} else {
		ctx = s.newRootContext(contextID)
	}

	s.rootContexts[contextID] = &rootContextState{
		context: ctx,
		httpCallbacks: map[uint32]*struct {
			callback        HttpCalloutCallBack
			callerContextID uint32
		}{},
	}
}

func (s *state) createStreamContext(contextID uint32, rootContextID uint32) {
	if _, ok := s.rootContexts[rootContextID]; !ok {
		panic("invalid root context id")
	}

	if _, ok := s.streams[contextID]; ok {
		panic("context id duplicated")
	}

	ctx := s.newStreamContext(rootContextID, contextID)
	s.contextIDToRootID[contextID] = rootContextID
	s.streams[contextID] = ctx
}

func (s *state) createHttpContext(contextID uint32, rootContextID uint32) {
	if _, ok := s.rootContexts[rootContextID]; !ok {
		panic("invalid root context id")
	}

	if _, ok := s.httpStreams[contextID]; ok {
		panic("context id duplicated")
	}

	ctx := s.newHttpContext(rootContextID, contextID)
	s.contextIDToRootID[contextID] = rootContextID
	s.httpStreams[contextID] = ctx
}

func (s *state) registerHttpCallOut(calloutID uint32, callback HttpCalloutCallBack) {
	r := s.rootContexts[s.contextIDToRootID[s.activeContextID]]
	r.httpCallbacks[calloutID] = &struct {
		callback        HttpCalloutCallBack
		callerContextID uint32
	}{callback: callback, callerContextID: s.activeContextID}
}

//go:inline
func (s *state) setActiveContextID(contextID uint32) {
	s.activeContextID = contextID
}
