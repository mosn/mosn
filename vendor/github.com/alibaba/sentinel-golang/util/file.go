// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"errors"
	"io"
	"os"
)

func FilePosition(file *os.File) (int64, error) {
	if file == nil {
		return 0, errors.New("null fd when retrieving file position")
	}
	return file.Seek(0, io.SeekCurrent)
}

func FileExists(name string) (b bool, err error) {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
	}
	// Propagates the error if the error is not FileNotExist error.
	return true, err
}

func CreateDirIfNotExists(dirname string) error {
	if _, err := os.Stat(dirname); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dirname, os.ModePerm)
		} else {
			return err
		}
	}
	return nil
}
