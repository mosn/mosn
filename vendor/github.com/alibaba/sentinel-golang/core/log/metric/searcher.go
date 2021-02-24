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

package metric

import (
	"encoding/binary"
	"io"
	"os"
	"sync"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

const offsetNotFound = -1

type DefaultMetricSearcher struct {
	reader MetricLogReader

	baseDir      string
	baseFilename string

	cachedPos *filePosition

	mux *sync.Mutex
}

type filePosition struct {
	metricFilename string
	idxFilename    string

	curOffsetInIdx uint64
	curSecInIdx    uint64

	// TODO: cache the idx file handle here?
}

func (s *DefaultMetricSearcher) FindByTimeAndResource(beginTimeMs uint64, endTimeMs uint64, resource string) ([]*base.MetricItem, error) {
	return s.searchOffsetAndRead(beginTimeMs, func(filenames []string, fileNo uint32, offset uint64) (items []*base.MetricItem, err error) {
		return s.reader.ReadMetricsByEndTime(filenames, fileNo, offset, beginTimeMs, endTimeMs, resource)
	})
}

func (s *DefaultMetricSearcher) FindFromTimeWithMaxLines(beginTimeMs uint64, maxLines uint32) ([]*base.MetricItem, error) {
	return s.searchOffsetAndRead(beginTimeMs, func(filenames []string, fileNo uint32, offset uint64) (items []*base.MetricItem, err error) {
		return s.reader.ReadMetrics(filenames, fileNo, offset, maxLines)
	})
}

func (s *DefaultMetricSearcher) searchOffsetAndRead(beginTimeMs uint64, doRead func([]string, uint32, uint64) ([]*base.MetricItem, error)) ([]*base.MetricItem, error) {
	filenames, err := listMetricFiles(s.baseDir, s.baseFilename)
	if err != nil {
		return nil, err
	}
	// Try to position the latest file index and offset from the cache (fast-path).
	// If cache is not up-to-date, we'll read from the initial position (offset 0 of the first file).
	offsetStart, fileNo, err := s.getOffsetStartAndFileIdx(filenames, beginTimeMs)
	if err != nil {
		logging.Warn("[searchOffsetAndRead] Failed to getOffsetStartAndFileIdx", "beginTimeMs", beginTimeMs, "err", err.Error())
	}
	fileAmount := uint32(len(filenames))
	for i := fileNo; i < fileAmount; i++ {
		filename := filenames[i]
		// Retrieve the start offset that is valid for given condition.
		// If offset = -1, it indicates that current file (i) does not satisfy the condition.
		offset, err := s.findOffsetToStart(filename, beginTimeMs, offsetStart)
		if err != nil {
			logging.Warn("[searchOffsetAndRead] Failed to findOffsetToStart, will try next file", "beginTimeMs", beginTimeMs,
				"filename", filename, "offsetStart", offsetStart, "err", err)
			continue
		}
		if offset >= 0 {
			// Read metric items from the offset of current file (number i).
			return doRead(filenames, i, uint64(offset))
		}
	}
	return make([]*base.MetricItem, 0), nil
}

func (s *DefaultMetricSearcher) getOffsetStartAndFileIdx(filenames []string, beginTimeMs uint64) (offsetInIdx uint64, i uint32, err error) {
	cacheOk, err := s.isPositionInTimeFor(beginTimeMs)
	if err != nil {
		return
	}
	if cacheOk {
		for j, v := range filenames {
			if v != s.cachedPos.metricFilename {
				i = uint32(j)
				offsetInIdx = s.cachedPos.curOffsetInIdx
				break
			}
		}
	}
	return
}

func (s *DefaultMetricSearcher) findOffsetToStart(filename string, beginTimeMs uint64, lastPos uint64) (int64, error) {
	s.cachedPos.idxFilename = ""
	s.cachedPos.metricFilename = ""

	idxFilename := formMetricIdxFileName(filename)
	if _, err := os.Stat(idxFilename); err != nil {
		return 0, err
	}
	beginSec := beginTimeMs / 1000
	file, err := os.Open(idxFilename)
	if err != nil {
		return 0, errors.Wrap(err, "failed to open metric idx file: "+idxFilename)
	}
	defer file.Close()

	// Set position to the offset recorded in the idx file
	_, err = file.Seek(int64(lastPos), io.SeekStart)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to fseek idx to offset %d", lastPos)
	}
	curPos, err := util.FilePosition(file)
	if err != nil {
		return 0, nil
	}
	s.cachedPos.curOffsetInIdx = uint64(curPos)
	var sec uint64 = 0
	var offset int64 = 0
	for {
		err = binary.Read(file, binary.BigEndian, &sec)
		if err != nil {
			if err == io.EOF {
				// EOF but offset hasn't been found yet, which indicates the expected position is not in current file
				return offsetNotFound, nil
			}
			return 0, err
		}
		if sec >= beginSec {
			break
		}
		err = binary.Read(file, binary.BigEndian, &offset)
		if err != nil {
			return 0, err
		}
		curPos, err := util.FilePosition(file)
		if err != nil {
			return 0, nil
		}
		s.cachedPos.curOffsetInIdx = uint64(curPos)
	}
	err = binary.Read(file, binary.BigEndian, &offset)
	if err != nil {
		return 0, err
	}
	// Cache the idx filename and position
	s.cachedPos.metricFilename = filename
	s.cachedPos.idxFilename = idxFilename
	s.cachedPos.curSecInIdx = sec

	return offset, nil
}

func (s *DefaultMetricSearcher) isPositionInTimeFor(beginTimeMs uint64) (bool, error) {
	if beginTimeMs/1000 < s.cachedPos.curSecInIdx {
		return false, nil
	}
	idxFilename := s.cachedPos.idxFilename
	if idxFilename == "" {
		return false, nil
	}
	if _, err := os.Stat(idxFilename); err != nil {
		return false, err
	}
	idxFile, err := openFileAndSeekTo(idxFilename, s.cachedPos.curOffsetInIdx)
	if err != nil {
		return false, err
	}
	defer idxFile.Close()

	var sec uint64
	err = binary.Read(idxFile, binary.BigEndian, &sec)
	if err != nil {
		return false, err
	}

	return sec == s.cachedPos.curSecInIdx, nil
}

func NewDefaultMetricSearcher(baseDir, baseFilename string) (MetricSearcher, error) {
	if baseDir == "" {
		return nil, errors.New("empty base directory")
	}
	if baseFilename == "" {
		return nil, errors.New("empty base filename pattern")
	}
	if baseDir[len(baseDir)-1] != os.PathSeparator {
		baseDir = baseDir + string(os.PathSeparator)
	}
	reader := newDefaultMetricLogReader()
	return &DefaultMetricSearcher{
		baseDir:      baseDir,
		baseFilename: baseFilename,
		reader:       reader,
		cachedPos:    &filePosition{},
		mux:          new(sync.Mutex),
	}, nil
}
