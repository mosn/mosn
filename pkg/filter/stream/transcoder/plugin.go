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

package transcoder

import (
	"mosn.io/api/extensions/transcoder"
	"mosn.io/mosn/pkg/log"
	goplugin "plugin"
)

type TranscoderGoPlugin struct {
	SrcPro string `json:"src_protocol,omitempty"`
	DstPro string `json:"dst_protocol,omitempty"`
	SoPath string `json:"so_path,omitempty"`
}

func (t *TranscoderGoPlugin) CreateTranscoder(listenerName string) {

	if t.SrcPro == "" || t.DstPro == "" || t.SoPath == "" {
		log.DefaultLogger.Errorf("[stream filter][transcoder] config could not be found, srcPro: %s,"+
			" dsrPro: %s, soPath: %s", t.SrcPro, t.DstPro, t.SoPath)
		return
	}

	name := listenerName + "_" + t.SrcPro + "_" + t.DstPro

	if GetTranscoder(name) != nil {
		return
	}

	p, err := goplugin.Open(t.SoPath)
	if err != nil {
		log.DefaultLogger.Errorf("[stream filter][transcoder] so file could not be load, soPath: %s, err: %v", t.SoPath, err)
		return
	}

	sym, err := p.Lookup("LoadTranscoder")
	if err != nil {
		log.DefaultLogger.Errorf("[stream filter][transcoder] so file look up error, soPath: %s, err: %v", t.SoPath, err)
		return
	}

	loadFunc := sym.(func() transcoder.Transcoder)
	transcoderSo := loadFunc()

	MustRegister(name, transcoderSo)
}
