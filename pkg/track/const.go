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

package track

type TrackPhase int

const MaxServedField = StreamSendFilter

const NoTrack TrackPhase = -1

const (
	// reserved fields
	ProtocolDecode          TrackPhase = iota // request and response
	StreamFilterBeforeRoute                   //
	MatchRoute                                // maybe reroute, or multiple steps
	StreamFilterAfterRoute
	LoadBalanceChooseHost // maybe rechoose host
	StreamFilterAfterChooseHost
	NetworkDataWrite // request and response
	StreamSendFilter
	// some extends fields
	// max phase
	MaxTrackPhase TrackPhase = 15
)

type TimestampPhase int

const (
	RequestStartTimestamp TimestampPhase = iota
	ResponseStartTimestamp
	MaxTimestampPhase
)

// TrackStartTimestamp is an alias for RequestStartTimestamp
// we think a mosn stream is begin when a request received
const TrackStartTimestamp TimestampPhase = RequestStartTimestamp
