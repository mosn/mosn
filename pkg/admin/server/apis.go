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
	"os"

	"io/ioutil"

	"sofastack.io/sofa-mosn/pkg/admin/store"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/metrics"
	"sofastack.io/sofa-mosn/pkg/metrics/sink/console"
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
		log.DefaultLogger.Errorf("[admin api] [config dump] invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if buf, err := store.Dump(); err == nil {
		log.DefaultLogger.Infof("[admin api] [config dump] config dump")
		w.WriteHeader(200)
		w.Write(buf)
	} else {
		log.DefaultLogger.Errorf("[admin api] [config dump] config failed, error: %v", err)
		w.WriteHeader(500)
		msg := fmt.Sprintf(errMsgFmt, "internal error")
		fmt.Fprint(w, msg)
	}
}

func statsDump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.DefaultLogger.Errorf("[admin api] [stats dump] invalid method: %s", r.Method)
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
		log.DefaultLogger.Errorf("[admin api] [update log level] invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Errorf("[admin api] [update log level] read body failed, error: %v", err)
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
	log.DefaultLogger.Errorf("[admin api] [update log level] update logger level failed with bad request data: %s", string(body))
	w.WriteHeader(http.StatusBadRequest) // 400
	msg := fmt.Sprintf(errMsgFmt, "update logger failed")
	fmt.Fprint(w, msg)
}

// post data:
// loggeer path
func enableLogger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.DefaultLogger.Errorf("[admin api] [enable logger] invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	loggerPath, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Errorf("[admin api] [enable logger] read body failed, error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	if !log.ToggleLogger(string(loggerPath), false) {
		log.DefaultLogger.Errorf("[admin api] [enable logger] enbale logger failed, logger: %s", string(loggerPath))
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
		log.DefaultLogger.Errorf("[admin api] [disable logger] invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	loggerPath, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.DefaultLogger.Errorf("[admin api] [disable logger] read body failed, error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf(errMsgFmt, "read body error")
		fmt.Fprint(w, msg)
		return
	}
	if !log.ToggleLogger(string(loggerPath), true) {
		log.DefaultLogger.Errorf("[admin api] [disable logger] disale logger failed, logger: %s", string(loggerPath))
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
		log.DefaultLogger.Errorf("[admin api] [get mosn state] invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	pid := os.Getpid()
	state := store.GetMosnState()
	msg := fmt.Sprintf("pid=%d&state=%d\n", pid, state)
	fmt.Fprint(w, msg)
}
