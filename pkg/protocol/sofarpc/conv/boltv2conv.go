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

package conv

import (
	"context"
	"reflect"
	"strconv"

	"errors"

	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
)

var (
	// BoltV2PropertyHeaders map the cmdkey and its data type
	BoltV2PropertyHeaders = make(map[string]reflect.Kind, 14)
	boltv2                = new(boltv2conv)
)

func init() {
	BoltV2PropertyHeaders[sofarpc.HeaderProtocolCode] = reflect.Uint8
	BoltV2PropertyHeaders[sofarpc.HeaderCmdType] = reflect.Uint8
	BoltV2PropertyHeaders[sofarpc.HeaderCmdCode] = reflect.Int16
	BoltV2PropertyHeaders[sofarpc.HeaderVersion] = reflect.Uint8
	BoltV2PropertyHeaders[sofarpc.HeaderReqID] = reflect.Uint32
	BoltV2PropertyHeaders[sofarpc.HeaderCodec] = reflect.Uint8
	BoltV2PropertyHeaders[sofarpc.HeaderClassLen] = reflect.Int16
	BoltV2PropertyHeaders[sofarpc.HeaderHeaderLen] = reflect.Int16
	BoltV2PropertyHeaders[sofarpc.HeaderContentLen] = reflect.Int
	BoltV2PropertyHeaders[sofarpc.HeaderTimeout] = reflect.Int
	BoltV2PropertyHeaders[sofarpc.HeaderRespStatus] = reflect.Int16
	BoltV2PropertyHeaders[sofarpc.HeaderRespTimeMills] = reflect.Int64
	BoltV2PropertyHeaders[sofarpc.HeaderVersion1] = reflect.Uint8
	BoltV2PropertyHeaders[sofarpc.HeaderSwitchCode] = reflect.Uint8

	sofarpc.RegisterConv(sofarpc.PROTOCOL_CODE_V2, boltv2)
}

type boltv2conv struct{}

func (b *boltv2conv) MapToCmd(ctx context.Context, headers map[string]string) (sofarpc.ProtoBasicCmd, error) {
	if len(headers) < 10 {
		return nil, errors.New("headers count not enough")
	}

	cmdV1, _ := boltv1.MapToCmd(ctx, headers)

	value := sofarpc.GetPropertyValue1(BoltV2PropertyHeaders, headers, sofarpc.HeaderVersion1)
	ver1 := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue1(BoltV2PropertyHeaders, headers, sofarpc.HeaderSwitchCode)
	switchcode := sofarpc.ConvertPropertyValueUint8(value)

	if cmdV2req, ok := cmdV1.(*sofarpc.BoltRequestCommand); ok {
		request := &sofarpc.BoltV2RequestCommand{
			BoltRequestCommand: *cmdV2req,
			Version1:           ver1,
			SwitchCode:         switchcode,
		}

		return request, nil
	} else if cmdV2res, ok := cmdV1.(*sofarpc.BoltResponseCommand); ok {
		response := &sofarpc.BoltV2ResponseCommand{
			BoltResponseCommand: *cmdV2res,
			Version1:            ver1,
			SwitchCode:          switchcode,
		}

		return response, nil
	} else {
		// todo RPC_HB
	}

	return nil, errors.New(sofarpc.InvalidCommandType)
}

func (b *boltv2conv) MapToFields(ctx context.Context, cmd sofarpc.ProtoBasicCmd) (map[string]string, error) {
	switch c := cmd.(type) {
	case *sofarpc.BoltV2RequestCommand:
		return mapReqV2ToFields(ctx, c)

	case *sofarpc.BoltV2ResponseCommand:
		return mapRespV2ToFields(ctx, c)
	}

	return nil, errors.New(sofarpc.InvalidCommandType)
}

func mapReqV2ToFields(ctx context.Context, req *sofarpc.BoltV2RequestCommand) (map[string]string, error) {
	headers, _ := mapReqToFields(ctx, &req.BoltRequestCommand)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(req.Version1), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(req.SwitchCode), 10)

	return headers, nil
}

func mapRespV2ToFields(ctx context.Context, resp *sofarpc.BoltV2ResponseCommand) (map[string]string, error) {
	headers, _ := mapRespToFields(ctx, &resp.BoltResponseCommand)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(resp.Version1), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(resp.SwitchCode), 10)

	return headers, nil
}
