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

package zipkin

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/openzipkin/zipkin-go/idgenerator"
	"github.com/openzipkin/zipkin-go/model"
)

func Test_B3(t *testing.T) {
	idgen := idgenerator.NewRandom128()

	for i := 0; i != 10; i++ {

		spanCtx := model.SpanContext{}
		spanCtx.TraceID = idgen.TraceID()
		spanCtx.ID = idgen.SpanID(spanCtx.TraceID)
		spanCtx.ParentID = &spanCtx.ID
		spanCtx.ID = idgen.SpanID(spanCtx.TraceID)

		t.Run("", func(t *testing.T) {
			tmp := http.Header{}
			InjectB3(tmp, spanCtx)
			got, err := ExtractB3(tmp)
			if err != nil {
				t.Errorf("ExtractB3() error: %v", err)
			}
			if !reflect.DeepEqual(got, &spanCtx) {
				t.Errorf("ExtractB3() = %v, want %v", got, &spanCtx)
			}
		})
	}
}
