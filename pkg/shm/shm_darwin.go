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
	"os"
	"syscall"
)

func Alloc(name string, size int) (*ShmSpan, error) {
	path := path(name)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	if err := f.Truncate(int64(size)); err != nil {
		return nil, err
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_WRITE, syscall.MAP_SHARED)

	if err != nil {
		return nil, err
	}

	return NewShmSpan(name, data), nil
}

func DeAlloc(span *ShmSpan) error {
	os.Remove(path(span.name))
	return syscall.Munmap(span.origin)
}
