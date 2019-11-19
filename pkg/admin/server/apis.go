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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"sofastack.io/sofa-mosn/pkg/admin/store"
	"sofastack.io/sofa-mosn/common/log"
	"sofastack.io/sofa-mosn/pkg/metrics"
	"sofastack.io/sofa-mosn/pkg/metrics/sink/console"
	"sofastack.io/sofa-mosn/pkg/types"
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

func help(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	buf.WriteString("support apis:\n")
	for key := range apiHandleFuncStore {
		if key != "/" { // do not show this pages
			buf.WriteString(key)
			buf.WriteRune('\n')
		}
	}
	w.Write(buf.Bytes())
}

func configDump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: invalid method: %s", "config dump", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if buf, err := store.Dump(); err == nil {
		log.DefaultLogger.Infof("[admin api] [config dump] config dump")
		w.WriteHeader(200)
		w.Write(buf)
	} else {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: %v", "config dump", err)
		w.WriteHeader(500)
		msg := fmt.Sprintf(errMsgFmt, "internal error")
		fmt.Fprint(w, msg)
	}
}

func statsDump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: invalid method: %s", "stats dump", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	log.DefaultLogger.Infof("[admin api]  [stats dump] stats dump")
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
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: invalid method: %s", "update log level", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: read body failed, %v", "update log level", err)
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	data := &LogLevelData{}
	if err := json.Unmarshal(body, data); err == nil {
		if level, ok := levelMap[data.LogLevel]; ok {
			if log.UpdateErrorLoggerLevel(data.LogPath, level) {
				log.DefaultLogger.Infof("[admin api] [update log level] update log: %s level as %s", data.LogPath, data.LogLevel)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "update logger success\n")
				return
			}
		}
	}
	log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, update logger level failed with bad request data: %s", "update log level", string(body))
	w.WriteHeader(http.StatusBadRequest) // 400
	msg := fmt.Sprintf(errMsgFmt, "update logger failed")
	fmt.Fprint(w, msg)
}

// post data:
// loggeer path
func enableLogger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: invalid method: %s", "enable logger", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	loggerPath, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: read body failed, %v", "enable logger", err)
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	if !log.ToggleLogger(string(loggerPath), false) {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: enbale %s logger failed", "enable logger", string(loggerPath))
		w.WriteHeader(http.StatusBadRequest) // 400
		msg := fmt.Sprintf(errMsgFmt, "enable logger failed")
		fmt.Fprint(w, msg)
		return
	}
	log.DefaultLogger.Infof("[admin api] [enable logger] enable logger %s", string(loggerPath))
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "enable logger success\n")
}

func disableLogger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: invalid method: %s", "disable logger", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	loggerPath, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: read body failed, %v", "disable logger", err)
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	if !log.ToggleLogger(string(loggerPath), true) {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: enbale %s logger failed", "disable logger", string(loggerPath))
		w.WriteHeader(http.StatusBadRequest) // 400
		msg := fmt.Sprintf(errMsgFmt, "disbale logger failed")
		fmt.Fprint(w, msg)
		return
	}
	log.DefaultLogger.Infof("[admin api] [disable logger] disable logger %s", string(loggerPath))
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "disable logger success\n")
}

// returns data
// pid=xxx&state=xxx
func getState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.DefaultLogger.Alertf(types.ErrorKeyAdmin, "api: %s, error: invalid method: %s", "get state", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	pid := os.Getpid()
	state := store.GetMosnState()
	msg := fmt.Sprintf("pid=%d&state=%d\n", pid, state)
	fmt.Fprint(w, msg)
}
