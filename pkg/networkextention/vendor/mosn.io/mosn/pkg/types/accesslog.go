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

package types

// DefaultAccessLogFormat provides a pre-defined format
const DefaultAccessLogFormat = "%start_time% %request_received_duration% %response_received_duration% %bytes_sent%" + " " +
	"%bytes_received% %protocol% %response_code% %duration% %response_flag% %response_code% %upstream_local_address%" + " " +
	"%downstream_local_address% %downstream_remote_address% %upstream_host%"
