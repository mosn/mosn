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

package sofa

import (
	"context"
	"errors"
	"os/user"
	"path"
	"sync"
	"time"

	"mosn.io/api"
	"mosn.io/pkg/log"
)

var PrintLog = true

// Tracer is default trace action
type Tracer struct {
	ingressLogger *log.Logger
	egressLogger  *log.Logger
	init          sync.Once
}

func NewTracer(config map[string]interface{}) (api.Tracer, error) {
	tracer := &Tracer{}
	if PrintLog {
		logPath := ""
		if value, ok := config["log_path"]; ok {
			if lp, ok := value.(string); ok {
				logPath = lp
			}
		}
		if err := tracer.InitLogger(logPath, "rpc-server-digest.log", "rpc-client-digest.log"); err != nil {
			return nil, err
		}

	}
	return tracer, nil
}

func (tracer *Tracer) InitLogger(root, ingress, egress string) (e error) {
	tracer.init.Do(func() {
		if root == "" {
			// get default log root
			usr, err := user.Current()
			if err != nil {
				e = err
				return
			}
			root = path.Join(usr.HomeDir, "/logs/tracelog/mosn/")
		}
		if ingress == "" || egress == "" {
			e = errors.New("trace logger file name cannot be empty")
			return
		}
		ingressLogger, err := log.GetOrCreateLogger(path.Join(root, ingress), nil)
		if err != nil {
			e = err
			return
		}
		egressLogger, err := log.GetOrCreateLogger(path.Join(root, egress), nil)
		if err != nil {
			e = err
			return
		}
		tracer.ingressLogger = ingressLogger
		tracer.egressLogger = egressLogger
	})
	return
}

func (tracer *Tracer) Start(ctx context.Context, frame interface{}, startTime time.Time) api.Span {
	return tracer.NewSpan(ctx, startTime)
}

func (tracer *Tracer) NewSpan(ctx context.Context, startTime time.Time) *SofaRPCSpan {
	return &SofaRPCSpan{
		ctx:           ctx,
		startTime:     startTime,
		ingressLogger: tracer.ingressLogger,
		egressLogger:  tracer.egressLogger,
	}
}
