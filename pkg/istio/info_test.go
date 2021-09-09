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
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPodLabels(t *testing.T) {
	IstioPodInfoPath = "/tmp/test/pod"
	os.RemoveAll(IstioPodInfoPath)
	err := os.MkdirAll(IstioPodInfoPath, 0755)
	require.Nil(t, err)
	data := []byte("labela=1\nlabelb=2\nlabelc=\"c\"")
	err = ioutil.WriteFile(path.Join(IstioPodInfoPath, "labels"), data, 0644)
	require.Nil(t, err)
	labels := GetPodLabels()
	require.Len(t, labels, 3)
	require.Equal(t, "1", labels["labela"])
	require.Equal(t, "2", labels["labelb"])
	require.Equal(t, "c", labels["labelc"])

}
