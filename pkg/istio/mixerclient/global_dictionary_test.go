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

package mixerclient

import (
	"testing"
)

func TestAddAttributeToGlobalList(t *testing.T) {
	AddAttributeToGlobalList("attra")
	AddAttributeToGlobalList("attrb")
	AddAttributeToGlobalList("attrc")

	g := GlobalList()
	l := len(g)
	if g[l-3] != "attra" || g[l-2] != "attrb" || g[l-1] != "attrc" {
		t.Errorf("expected attra, attrb, attrc")
	}

}
