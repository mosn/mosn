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

package server

import (
	"fmt"
	"net/http"

	"io/ioutil"

	"github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/metrics"
	"github.com/alipay/sofa-mosn/pkg/metrics/sink/console"
)

var levelMap = map[string]log.Level{
	"FATAL": log.FATAL,
	"ERROR": log.ERROR,
	"WARN":  log.WARN,
	"INFO":  log.INFO,
	"DEBUG": log.DEBUG,
	"TRACE": log.TRACE,
}

const errMsgFmt = `{
	"error": "%s"
}
`

func configDump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if buf, err := store.Dump(); err == nil {
		w.WriteHeader(200)
		w.Write(buf)
	} else {
		w.WriteHeader(500)
		msg := fmt.Sprintf(errMsgFmt, "internal error")
		fmt.Fprint(w, msg)
		log.DefaultLogger.Errorf("Admin API: ConfigDump failed, cause by %s", err)
	}
}

func statsDump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(200)
	sink := console.NewConsoleSink()
	sink.Flush(w, metrics.GetAll())
}

// update log level
type LogLevelData struct {
	LogPath  string `json:"log_path"`
	LogLevel string `json:"log_level"`
}

func updateLogLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	data := &LogLevelData{}
	if err := json.Unmarshal(body, data); err == nil {
		if level, ok := levelMap[data.LogLevel]; ok {
			if log.UpdateErrorLoggerLevel(data.LogPath, level) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "update logger success\n")
				return
			}
		}
	}
	w.WriteHeader(http.StatusBadRequest) // 400
	msg := fmt.Sprintf(errMsgFmt, "update logger failed")
	fmt.Fprint(w, msg)
	log.DefaultLogger.Errorf("Admin API: update logger level failed with bad request data: %s", string(body))
}

// post data:
// loggeer path
func enableLogger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	loggerPath, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	if !log.ToggleLogger(string(loggerPath), false) {
		w.WriteHeader(http.StatusBadRequest) // 400
		msg := fmt.Sprintf(errMsgFmt, "enable logger failed")
		fmt.Fprint(w, msg)
		log.DefaultLogger.Errorf("Admin API: enbale logger failed, logger: %s", string(loggerPath))
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "enable logger success\n")
}

func disableLogger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	loggerPath, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	if !log.ToggleLogger(string(loggerPath), true) {
		w.WriteHeader(http.StatusBadRequest) // 400
		msg := fmt.Sprintf(errMsgFmt, "disbale logger failed")
		fmt.Fprint(w, msg)
		log.DefaultLogger.Errorf("Admin API: disable logger failed, logger: %s", string(loggerPath))
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "disable logger success\n")
}
