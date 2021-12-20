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
	"encoding/json"
	"errors"
	goplugin "plugin"

	"mosn.io/api/extensions/transcoder"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

func init() {
	v2.RegisterParseExtendConfig("transcoder_plugin", OnTranscoderPluginParsed)
}

type TranscoderGoPlugin struct {
	SrcPro string `json:"src_protocol,omitempty"`
	DstPro string `json:"dst_protocol,omitempty"`
	SoPath string `json:"so_path,omitempty"`
}

func (t *TranscoderGoPlugin) GetType() string {
	return t.SrcPro + "_" + t.DstPro
}

func (t *TranscoderGoPlugin) CreateTranscoder() error {
	if t.SrcPro == "" || t.DstPro == "" || t.SoPath == "" {
		log.DefaultLogger.Errorf("[stream filter][transcoder] config could not be found, srcPro: %s,"+
			" dsrPro: %s, soPath: %s", t.SrcPro, t.DstPro, t.SoPath)
		return errors.New("illegal parameter")
	}

	name := t.GetType()
	if GetTranscoderFactory(name) != nil {
		return nil
	}

	p, err := goplugin.Open(t.SoPath)
	if err != nil {
		log.DefaultLogger.Errorf("[stream filter][transcoder] so file could not be load, soPath: %s, err: %v", t.SoPath, err)
		return err
	}

	sym, err := p.Lookup("LoadTranscoderFactory")
	if err != nil {
		log.DefaultLogger.Errorf("[stream filter][transcoder] so file look up error, soPath: %s, err: %v", t.SoPath, err)
		return err
	}
	loadFunc, ok := sym.(func(cfg map[string]interface{}) transcoder.Transcoder)
	if !ok {
		err := errors.New("type not support")
		log.DefaultLogger.Errorf("[stream filter][transcoder] type change failed error, soPath: %s, err: %v", t.SoPath, err)
		return err
	}
	MustRegister(name, loadFunc)
	return nil
}

func OnTranscoderPluginParsed(data json.RawMessage) error {
	cfg := &transcodeGoPluginConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	if cfg.Transcoders != nil {
		for _, transcoder := range cfg.Transcoders {
			transcoder.CreateTranscoder()
		}
	}
	return nil
}
