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
	Header     string = "sw6"
	splitToken string = "-"
	idToken    string = "."
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
	TraceID                 []int64
	ParentSegmentID         []int64
	ParentSpanID            int32
	ParentServiceInstanceID int32
	EntryServiceInstanceID  int32
	NetworkAddressID        int32
	EntryEndpointID         int32
	ParentEndpointID        int32
	Sample                  int8
	NetworkAddress          string
	EntryEndpoint           string
	ParentEndpoint          string
}

// DecodeSW6 converts string header to SpanContext
func (tc *SpanContext) DecodeSW6(header string) error {
	if header == "" {
		return errEmptyHeader
	}
	hh := strings.Split(header, splitToken)
	if len(hh) < 7 {
		return errors.WithMessagef(errInsufficientHeaderEntities, "header string: %s", header)
	}
	sample, err := strconv.ParseInt(hh[0], 10, 8)
	if err != nil {
		return errors.Errorf("str to int8 error %s", hh[0])
	}
	tc.Sample = int8(sample)
	tc.TraceID, err = stringConvertGlobalID(hh[1])
	if err != nil {
		return errors.Wrap(err, "trace id parse error")
	}
	tc.ParentSegmentID, err = stringConvertGlobalID(hh[2])
	if err != nil {
		return errors.Wrap(err, "parent segment id parse error")
	}
	tc.ParentSpanID, err = stringConvertInt32(hh[3])
	if err != nil {
		return errors.Wrap(err, "parent span id parse error")
	}
	tc.ParentServiceInstanceID, err = stringConvertInt32(hh[4])
	if err != nil {
		return errors.Wrap(err, "parent service instance id parse error")
	}
	tc.EntryServiceInstanceID, err = stringConvertInt32(hh[5])
	if err != nil {
		return errors.Wrap(err, "entry service instance id parse error")
	}
	tc.NetworkAddress, tc.NetworkAddressID, err = decodeBase64(hh[6])
	if err != nil {
		return errors.Wrap(err, "network address parse error")
	}
	if len(hh) < 9 {
		return nil
	}
	tc.EntryEndpoint, tc.EntryEndpointID, err = decodeBase64(hh[7])
	if err != nil {
		return errors.Wrap(err, "entry endpoint parse error")
	}
	tc.ParentEndpoint, tc.ParentEndpointID, err = decodeBase64(hh[8])
	if err != nil {
		return errors.Wrap(err, "parent endpoint parse error")
	}
	return nil
}

// EncodeSW6 converts SpanContext to string header
func (tc *SpanContext) EncodeSW6() string {
	return strings.Join([]string{
		fmt.Sprint(tc.Sample),
		globalIDConvertString(tc.TraceID),
		globalIDConvertString(tc.ParentSegmentID),
		fmt.Sprint(tc.ParentSpanID),
		fmt.Sprint(tc.ParentServiceInstanceID),
		fmt.Sprint(tc.EntryServiceInstanceID),
		encodeCompressedField(tc.NetworkAddressID, tc.NetworkAddress),
		encodeCompressedField(tc.EntryEndpointID, tc.EntryEndpoint),
		encodeCompressedField(tc.ParentEndpointID, tc.ParentEndpoint),
	}, "-")
}

func stringConvertGlobalID(str string) ([]int64, error) {
	idStr, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, errors.Wrapf(err, "decode id error %s", str)
	}
	ss := strings.Split(string(idStr), idToken)
	if len(ss) < 3 {
		return nil, errors.Errorf("decode id entities error %s", string(idStr))
	}
	ii := make([]int64, len(ss))
	for i, s := range ss {
		ii[i], err = strconv.ParseInt(s, 0, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "convert id error convert id entities to int32 error %s", s)
		}
	}
	return ii, nil
}

func stringConvertInt32(str string) (int32, error) {
	i, err := strconv.ParseInt(str, 0, 32)
	return int32(i), err
}

func decodeBase64(str string) (string, int32, error) {
	ret, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", 0, err
	}
	retStr := string(ret)
	if strings.HasPrefix(retStr, "#") {
		return retStr[1:], 0, nil
	}
	i, err := strconv.ParseInt(retStr, 0, 32)
	if err != nil {
		return "", 0, err
	}
	return "", int32(i), nil
}

func globalIDConvertString(id []int64) string {
	ii := make([]string, len(id))
	for i, v := range id {
		ii[i] = fmt.Sprint(v)
	}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(ii, ".")))
}

func encodeCompressedField(id int32, text string) string {
	if id != 0 {
		return base64.StdEncoding.EncodeToString([]byte(fmt.Sprint(id)))
	}
	return base64.StdEncoding.EncodeToString([]byte("#" + text))
}
