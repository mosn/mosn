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
package buffer

import (
	"context"
	"testing"
)

func Test_ByteBuffer(t *testing.T) {
	ctx := NewBufferPoolContext(context.Background())
	b := GetBytesByContext(ctx, 1000)
	if len(*b) != 1000 {
		t.Errorf("bytes len should %d", 1000)
	}
	if cap(*b) != 1024 {
		t.Errorf("bytes cap should %d", 1024)
	}
	bv := PoolContext(ctx)
	bv.Give()
}
