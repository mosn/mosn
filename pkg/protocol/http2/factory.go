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

package http2

import (
	"mosn.io/api"

	"mosn.io/mosn/pkg/module/http2"
)

func ServerProto(sc *http2.XServerConn) api.Protocol {
	return &serverCodec{sc: sc}
}

func ClientProto(cc *http2.MClientConn) api.Protocol {
	return &clientCodec{cc: cc}
}
