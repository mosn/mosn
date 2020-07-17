// Licensed to SkyAPM org under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. SkyAPM org licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package propagation holds the required function signatures for Injection and
Extraction. It also contains decoder and encoder of SkyWalking propagation protocol.
*/
package propagation

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	Header     string = "sw8"
	headerLen  int    = 8
	splitToken string = "-"
)

var (
	errEmptyHeader                = errors.New("empty header")
	errInsufficientHeaderEntities = errors.New("insufficient header entities")
)

// Extractor is a tool specification which define how to
// extract trace parent context from propagation context
type Extractor func() (string, error)

// Injector is a tool specification which define how to
// inject trace context into propagation context
type Injector func(header string) error

// SpanContext defines propagation specification of SkyWalking
type SpanContext struct {
	TraceID               string `json:"trace_id"`
	ParentSegmentID       string `json:"parent_segment_id"`
	ParentService         string `json:"parent_service"`
	ParentServiceInstance string `json:"parent_service_instance"`
	ParentEndpoint        string `json:"parent_endpoint"`
	AddressUsedAtClient   string `json:"address_used_at_client"`
	ParentSpanID          int32  `json:"parent_span_id"`
	Sample                int8   `json:"sample"`
}

// DecodeSW6 converts string header to SpanContext
func (tc *SpanContext) DecodeSW8(header string) error {
	if header == "" {
		return errEmptyHeader
	}
	hh := strings.Split(header, splitToken)
	if len(hh) < headerLen {
		return errors.WithMessagef(errInsufficientHeaderEntities, "header string: %s", header)
	}
	sample, err := strconv.ParseInt(hh[0], 10, 8)
	if err != nil {
		return errors.Errorf("str to int8 error %s", hh[0])
	}
	tc.Sample = int8(sample)
	tc.TraceID, err = decodeBase64(hh[1])
	if err != nil {
		return errors.Wrap(err, "trace id parse error")
	}
	tc.ParentSegmentID, err = decodeBase64(hh[2])
	if err != nil {
		return errors.Wrap(err, "parent segment id parse error")
	}
	tc.ParentSpanID, err = stringConvertInt32(hh[3])
	if err != nil {
		return errors.Wrap(err, "parent span id parse error")
	}
	tc.ParentService, err = decodeBase64(hh[4])
	if err != nil {
		return errors.Wrap(err, "parent service parse error")
	}
	tc.ParentServiceInstance, err = decodeBase64(hh[5])
	if err != nil {
		return errors.Wrap(err, "parent service instance parse error")
	}
	tc.ParentEndpoint, err = decodeBase64(hh[6])
	if err != nil {
		return errors.Wrap(err, "parent endpoint parse error")
	}
	tc.AddressUsedAtClient, err = decodeBase64(hh[7])
	if err != nil {
		return errors.Wrap(err, "network address parse error")
	}
	return nil
}

// EncodeSW6 converts SpanContext to string header
func (tc *SpanContext) EncodeSW8() string {
	return strings.Join([]string{
		fmt.Sprint(tc.Sample),
		encodeBase64(tc.TraceID),
		encodeBase64(tc.ParentSegmentID),
		fmt.Sprint(tc.ParentSpanID),
		encodeBase64(tc.ParentService),
		encodeBase64(tc.ParentServiceInstance),
		encodeBase64(tc.ParentEndpoint),
		encodeBase64(tc.AddressUsedAtClient),
	}, "-")
}

func stringConvertInt32(str string) (int32, error) {
	i, err := strconv.ParseInt(str, 0, 32)
	return int32(i), err
}

func decodeBase64(str string) (string, error) {
	ret, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}

func encodeBase64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}
