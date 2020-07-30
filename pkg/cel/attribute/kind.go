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

package attribute

type Kind uint8

const (
	// Invalid, default value.
	VALUE_TYPE_UNSPECIFIED Kind = iota
	// An undiscriminated variable-length string.
	STRING
	// An undiscriminated 64-bit signed integer.
	INT64
	// An undiscriminated 64-bit floating-point value.
	DOUBLE
	// An undiscriminated boolean value.
	BOOL
	// A point in time.
	TIMESTAMP
	// An IP address.
	IP_ADDRESS
	// An email address.
	EMAIL_ADDRESS
	// A URI.
	URI
	// A DNS name.
	DNS_NAME
	// A span between two points in time.
	DURATION
	// A map string -> string, typically used by headers.
	STRING_MAP

	// mosn context
	MOSN_CTX
)

func (k Kind) String() string {
	out, ok := KindName[k]
	if ok {
		return out
	}
	return VALUE_TYPE_UNSPECIFIED.String()
}

var KindName = map[Kind]string{
	VALUE_TYPE_UNSPECIFIED: "VALUE_TYPE_UNSPECIFIED",
	STRING:                 "STRING",
	INT64:                  "INT64",
	DOUBLE:                 "DOUBLE",
	BOOL:                   "BOOL",
	TIMESTAMP:              "TIMESTAMP",
	IP_ADDRESS:             "IP_ADDRESS",
	EMAIL_ADDRESS:          "EMAIL_ADDRESS",
	URI:                    "URI",
	DNS_NAME:               "DNS_NAME",
	DURATION:               "DURATION",
	STRING_MAP:             "STRING_MAP",
	MOSN_CTX:               "MOSN_CTX",
}
var KindValue = map[string]Kind{}

func init() {
	for k, v := range KindName {
		KindValue[v] = k
	}
}
