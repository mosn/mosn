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

package shm

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"mosn.io/mosn/pkg/types"
)

var (
	errNotEnough = errors.New("span capacity is not enough")
)

func path(name string) string {
	return types.MosnConfigPath + string(os.PathSeparator) + fmt.Sprintf("mosn_shm_%s", name)
}

// check if given path match the required size
// return error if path exists and size not match
func checkConsistency(path string, size int) error {
	if info, err := os.Stat(path); err == nil {
		if info.Size() != int64(size) {
			return errors.New(fmt.Sprintf("mmap target path %s exists and its size %d mismatch %d", path, info.Size(), size))
		}
	}
	return nil
}

type ShmSpan struct {
	origin []byte
	name   string

	data   uintptr
	offset int
	size   int
}

func NewShmSpan(name string, data []byte) *ShmSpan {
	return &ShmSpan{
		name:   name,
		origin: data,
		data:   uintptr(unsafe.Pointer(&data[0])),
		size:   len(data),
	}
}

func (s *ShmSpan) Alloc(size int) (uintptr, error) {
	if s.offset+size > s.size {
		return 0, errNotEnough
	}

	ptr := s.data + uintptr(s.offset)
	s.offset += size
	return ptr, nil
}

func (s *ShmSpan) Data() uintptr {
	return s.data
}

func (s *ShmSpan) Origin() []byte {
	return s.origin
}
