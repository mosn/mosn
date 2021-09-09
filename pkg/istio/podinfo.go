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

package istio

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const podLabelsSeparator = "="

var (
	// IstioPodInfoPath Sidecar mountPath
	IstioPodInfoPath = "/etc/istio/pod"

	labels = make(map[string]string)

	once sync.Once
)

// GetPodLabels return podLabels map
func GetPodLabels() map[string]string {
	once.Do(func() {
		// Add pod labels into nodeInfo
		podLabelsPath := fmt.Sprintf("%s/labels", IstioPodInfoPath)
		if _, err := os.Stat(podLabelsPath); err == nil || os.IsExist(err) {
			f, err := os.Open(podLabelsPath)
			if err == nil {
				defer f.Close()

				br := bufio.NewReader(f)
				for {
					l, _, e := br.ReadLine()
					if e == io.EOF {
						break
					}
					// group="blue"
					keyValueSep := strings.SplitN(strings.ReplaceAll(string(l), "\"", ""), podLabelsSeparator, 2)
					if len(keyValueSep) != 2 {
						continue
					}
					labels[keyValueSep[0]] = keyValueSep[1]
				}
			}
		}
	})

	return labels
}
