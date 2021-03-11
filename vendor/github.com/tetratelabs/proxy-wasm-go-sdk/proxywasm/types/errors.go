// Copyright 2020 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"errors"
	"strconv"
)

var (
	ErrorStatusNotFound    = errors.New("error status returned by host: not found")
	ErrorStatusBadArgument = errors.New("error status returned by host: bad argument")
	ErrorStatusEmpty       = errors.New("error status returned by host: empty")
	ErrorStatusCasMismatch = errors.New("error status returned by host: cas mismatch")
	ErrorInternalFailure   = errors.New("error status returned by host: internal failure")
)

//go:inline
func StatusToError(status Status) error {
	switch status {
	case StatusOK:
		return nil
	case StatusNotFound:
		return ErrorStatusNotFound
	case StatusBadArgument:
		return ErrorStatusBadArgument
	case StatusEmpty:
		return ErrorStatusEmpty
	case StatusCasMismatch:
		return ErrorStatusCasMismatch
	case StatusInternalFailure:
		return ErrorInternalFailure
	}
	return errors.New("unknown status code: " + strconv.Itoa(int(status)))
}
