/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package stream

import (
	"context"
	"errors"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

var (
	FAILED = errors.New("FAILED")
	EAGAIN = errors.New("AGAIN")
)

func CreateServerStreamConnection(context context.Context, prot api.ProtocolName, connection api.Connection,
	callbacks types.ServerStreamConnectionEventListener) types.ServerStreamConnection {

	if ssc, ok := protocol.GetProtocolStreamFactory(prot); ok {
		return ssc.CreateServerStream(context, connection, callbacks)
	}

	return nil
}

// SelectStreamFactoryProtocol match the protocol.
// if scopes is nil, match all the registered protocols
// if scopes is not nil, match the protocol in the scopes
func SelectStreamFactoryProtocol(ctx context.Context, prot string, peek []byte, scopes []api.ProtocolName) (types.ProtocolName, error) {
	var err error
	var again bool
	streamFactories := protocol.GetProtocolStreamFactories(scopes)
	for p, factory := range streamFactories {
		err = factory.ProtocolMatch(ctx, prot, peek)
		if err == nil {
			return p, nil
		}
		if err == EAGAIN {
			again = true
		}
	}
	if again {
		return "", EAGAIN
	}
	return "", FAILED
}
