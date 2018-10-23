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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/istio/utils"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/gogo/protobuf/jsonpb"
	"istio.io/api/mixer/v1"
)

func init() {
	log.InitDefaultLogger("", log.DEBUG)
}

const (
	kAttributes = `{ "words":["JWT-Token"],"strings":{"2":127,"6":101},"int64s":{"1":"35","8":"8080"},"doubles":{"78":99.9},"bools":{"71":true},"timestamps":{"132":"1754-08-30T22:43:41.128654848Z"},"durations":{"29":"5s"},"bytes":{"0":"dGV4dC9odG1sOyBjaGFyc2V0PXV0Zi04"},"stringMaps":{"15":{"entries":{"50":0,"58":104}}}}`
	kReportAttributes = `{"attributes":[{"words":[],"strings":{"2":127,"6":101},"int64s":{"1":"35","8":"8080"},"doubles":{"78":99.9},"bools":{"71":true},"timestamps":{"132":"1754-08-30T22:43:41.128654848Z"},"durations":{"29":"5s"},"bytes":{"0":"dGV4dC9odG1sOyBjaGFyc2V0PXV0Zi04"},"stringMaps":{"15":{"entries":{"50":0,"58":104}}}},{"words":[],"strings":{"2":127,"6":101},"int64s":{"1":"135","8":"8080","27":"111"},"doubles":{"78":123.99},"bools":{"71":false},"timestamps":{"132":"1754-08-30T22:43:41.128654848Z"},"durations":{"29":"5s"},"bytes":{"0":"dGV4dC9odG1sOyBjaGFyc2V0PXV0Zi04"},"stringMaps":{"15":{"entries":{"32":90,"58":104}}}},{"words":[],"strings":{"2":127,"6":101},"int64s":{"1":"135","8":"8080"},"doubles":{"78":123.99},"bools":{"71":false},"timestamps":{"132":"1754-08-30T22:43:41.128654848Z"},"durations":{"29":"5s"},"bytes":{"0":"dGV4dC9odG1sOyBjaGFyc2V0PXV0Zi04"},"stringMaps":{"15":{"entries":{"32":90,"58":104}}}}],"defaultWords":["JWT-Token"],"globalWordCount":202}`
)

func initTestAttributes(attributes *v1.Attributes) {
	builder := utils.NewAttributesBuilder(attributes)

	builder.AddString("source.name", "connection.received.bytes_total");
	builder.AddBytes("source.ip", []byte("text/html; charset=utf-8"));
	builder.AddDouble("range", 99.9);
	builder.AddInt64("source.port", 35);
	builder.AddBool("keep-alive", true);
	builder.AddString("source.user", "x-http-method-override");
	builder.AddInt64("target.port", 8080);

	builder.AddTimestamp("context.timestamp", time.Time{})
	builder.AddDuration("response.duration", time.Second * 5)

	// JWT-token is only word not in the global dictionary.
	string_map := map[string]string {
		"authorization": "JWT-Token",
		"content-type": "application/json",
	}

	builder.AddStringMap("request.headers", protocol.CommonHeader(string_map));
}

func TestCompress(t *testing.T) {
	attributes := v1.Attributes {
		Attributes:make(map[string]*v1.Attributes_AttributeValue, 0),
	}

	initTestAttributes(&attributes)

	compressor := NewAttributeCompressor()
	pb := NewCompressAttributes()
	compressor.Compress(&attributes, &pb)

	mar := jsonpb.Marshaler{}
	str, _ := mar.MarshalToString(&pb)
	fmt.Printf("attributes: %s\n", string(str))
	fmt.Printf("attributes1: %s\n", strings.TrimSpace(kAttributes))
	if str != strings.TrimSpace(kAttributes) {
		t.Fatalf("not equal")
	}
}

func TestBatchCompress(t *testing.T) {
	attributes := v1.Attributes {
		Attributes:make(map[string]*v1.Attributes_AttributeValue, 0),
	}

	compressor := NewAttributeCompressor()
	initTestAttributes(&attributes)
	batchCompressor := NewBatchCompressor(compressor.globalDict)

	batchCompressor.Add(&attributes)

	// modify some attributes
	builder := utils.NewAttributesBuilder(&attributes)
	builder.AddDouble("range", 123.99);
	builder.AddInt64("source.port", 135);
	builder.AddInt64("response.size", 111);
	builder.AddBool("keep-alive", false);
	string_map := protocol.CommonHeader{
		"content-type": "application/json",
		":method": "GET",
	}
	builder.AddStringMap("request.headers", string_map)

	// Batch the second one with added attributes
	batchCompressor.Add(&attributes)

	// remove a key
	delete(attributes.Attributes, "response.size")
	// Batch the third with a removed attribute.
	batchCompressor.Add(&attributes)

	pb := batchCompressor.Finish()

	mar := jsonpb.Marshaler{}
	str, _ := mar.MarshalToString(pb)

	fmt.Printf("attributes: %s\n", string(str))
	if str != kReportAttributes {
		t.Fatalf("not equal")
	}
}
