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
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

type DefaultMetricLogWriter struct {
	baseDir      string
	baseFilename string

	maxSingleSize uint64
	maxFileAmount uint32

	timezoneOffsetSec int64
	latestOpSec       int64

	curMetricFile    *os.File
	curMetricIdxFile *os.File

	metricOut *bufio.Writer
	idxOut    *bufio.Writer

	mux *sync.RWMutex
}

func (d *DefaultMetricLogWriter) Write(ts uint64, items []*base.MetricItem) error {
	if len(items) == 0 {
		return nil
	}
	if ts <= 0 {
		return errors.New(fmt.Sprintf("%s: %d", "Invalid timestamp: ", ts))
	}
	if d.curMetricFile == nil || d.curMetricIdxFile == nil {
		return errors.New("file handle not initialized")
	}
	// Update all metric items to the given timestamp.
	for _, item := range items {
		item.Timestamp = ts
	}

	d.mux.Lock()
	defer d.mux.Unlock()

	timeSec := int64(ts / 1000)
	if timeSec < d.latestOpSec {
		// ignore
		return nil
	}
	if timeSec > d.latestOpSec {
		pos, err := util.FilePosition(d.curMetricFile)
		if err != nil {
			return errors.Wrap(err, "cannot get current pos of the metric file")
		}
		if err = d.writeIndex(timeSec, pos); err != nil {
			return errors.Wrap(err, "cannot write metric idx file")
		}
		if d.isNewDay(d.latestOpSec, timeSec) {
			if err = d.rollToNextFile(ts); err != nil {
				return errors.Wrap(err, "failed to roll the metric log")
			}
		}
	}
	// Write and flush
	if err := d.writeItemsAndFlush(items); err != nil {
		return errors.Wrap(err, "failed to write and flush metric items")
	}
	if err := d.rollFileIfSizeExceeded(ts); err != nil {
		return errors.Wrap(err, "failed to pre-check the rolling condition of metric logs")
	}
	if timeSec > d.latestOpSec {
		// Update the latest timeSec.
		d.latestOpSec = timeSec
	}

	return nil
}

func (d *DefaultMetricLogWriter) Close() error {
	d.mux.Lock()
	defer d.mux.Unlock()

	if d.curMetricIdxFile != nil {
		d.curMetricIdxFile.Close()
	}
	if d.curMetricFile != nil {
		return d.curMetricFile.Close()
	}
	return nil
}

func (d *DefaultMetricLogWriter) writeItemsAndFlush(items []*base.MetricItem) error {
	for _, item := range items {
		s, err := item.ToFatString()
		if err != nil {
			logging.Warn("[writeItemsAndFlush] Failed to convert MetricItem to string", "resourceName", item.Resource, "err", err.Error())
			continue
		}

		// Append the LF line separator.
		bs := []byte(s + "\n")
		_, err = d.metricOut.Write(bs)
		if err != nil {
			return nil
		}
	}
	return d.metricOut.Flush()
}

func (d *DefaultMetricLogWriter) rollFileIfSizeExceeded(time uint64) error {
	if d.curMetricFile == nil {
		return nil
	}
	stat, err := d.curMetricFile.Stat()
	if err != nil {
		return err
	}
	if uint64(stat.Size()) >= d.maxSingleSize {
		return d.rollToNextFile(time)
	}
	return nil
}

func (d *DefaultMetricLogWriter) rollToNextFile(time uint64) error {
	newFilename, err := d.nextFileNameOfTime(time)
	if err != nil {
		return err
	}
	return d.closeCurAndNewFile(newFilename)
}

func (d *DefaultMetricLogWriter) writeIndex(time, offset int64) error {
	out := d.idxOut
	if out == nil {
		return errors.New("index buffered writer not ready")
	}

	// Use BigEndian here to keep consistent with DataOutputStream in Java.
	err := binary.Write(out, binary.BigEndian, time)
	if err != nil {
		return err
	}
	err = binary.Write(out, binary.BigEndian, offset)
	if err != nil {
		return err
	}
	return out.Flush()
}

func (d *DefaultMetricLogWriter) removeDeprecatedFiles() error {
	files, err := listMetricFiles(d.baseDir, d.baseFilename)
	if err != nil || len(files) == 0 {
		return err
	}
	amountToRemove := len(files) - int(d.maxFileAmount) + 1
	for i := 0; i < amountToRemove; i++ {
		filename := files[i]
		idxFilename := formMetricIdxFileName(filename)
		err = os.Remove(filename)
		if err != nil {
			logging.Error(err, "Failed to remove metric log file in DefaultMetricLogWriter.removeDeprecatedFiles()", "filename", filename)
		} else {
			logging.Info("[MetricWriter] Metric log file removed in DefaultMetricLogWriter.removeDeprecatedFiles()", "filename", filename)
		}

		err = os.Remove(idxFilename)
		if err != nil {
			logging.Error(err, "Failed to remove metric log file in DefaultMetricLogWriter.removeDeprecatedFiles()", "idxFilename", idxFilename)
		} else {
			logging.Info("[MetricWriter] Metric index file removed", "idxFilename", idxFilename)
		}
	}
	return err
}

func (d *DefaultMetricLogWriter) nextFileNameOfTime(time uint64) (string, error) {
	dateStr := util.FormatDate(time)
	filePattern := d.baseFilename + "." + dateStr
	list, err := listMetricFilesConditional(d.baseDir, filePattern, func(fn string, p string) bool {
		return strings.Contains(fn, p)
	})
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return filepath.Join(d.baseDir, filePattern), nil
	}
	last := list[len(list)-1]
	var n uint32 = 0
	items := strings.Split(last, ".")
	if len(items) > 0 {
		v, err := strconv.ParseUint(items[len(items)-1], 10, 32)
		if err == nil {
			n = uint32(v)
		}
	}
	return filepath.Join(d.baseDir, fmt.Sprintf("%s.%d", filePattern, n+1)), nil
}

func (d *DefaultMetricLogWriter) closeCurAndNewFile(filename string) error {
	err := d.removeDeprecatedFiles()
	if err != nil {
		return err
	}

	if d.curMetricFile != nil {
		if err = d.curMetricFile.Close(); err != nil {
			logging.Error(err, "Failed to close metric log file in DefaultMetricLogWriter.closeCurAndNewFile()", "curMetricFile", d.curMetricFile.Name())
		}
	}
	if d.curMetricIdxFile != nil {
		if err = d.curMetricIdxFile.Close(); err != nil {
			logging.Error(err, "Failed to close metric index file in DefaultMetricLogWriter.closeCurAndNewFile()", "curMetricIdxFile", d.curMetricIdxFile.Name())
		}
	}
	// Create new metric log file, whether it exists or not.
	mf, err := os.Create(filename)
	if err != nil {
		return err
	}
	logging.Info("[MetricWriter] New metric log file created", "filename", filename)

	idxFile := formMetricIdxFileName(filename)
	mif, err := os.Create(idxFile)
	if err != nil {
		return err
	}
	logging.Info("[MetricWriter] New metric log index file created", "idxFile", idxFile)

	d.curMetricFile = mf
	d.metricOut = bufio.NewWriter(mf)

	d.curMetricIdxFile = mif
	d.idxOut = bufio.NewWriter(mif)

	return nil
}

func (d *DefaultMetricLogWriter) initialize() error {
	// Create the dir if not exists.
	err := util.CreateDirIfNotExists(d.baseDir)
	if err != nil {
		return err
	}

	if d.curMetricFile != nil {
		return nil
	}
	ts := util.CurrentTimeMillis()
	if err := d.rollToNextFile(ts); err != nil {
		return errors.Wrap(err, "failed to initialize metric log writer")
	}
	d.latestOpSec = int64(ts / 1000)
	return nil
}

func (d *DefaultMetricLogWriter) isNewDay(lastSec, sec int64) bool {
	prevDayTs := (lastSec + d.timezoneOffsetSec) / 86400
	newDayTs := (sec + d.timezoneOffsetSec) / 86400
	return newDayTs > prevDayTs
}

func NewDefaultMetricLogWriter(maxSize uint64, maxFileAmount uint32) (MetricLogWriter, error) {
	return NewDefaultMetricLogWriterOfApp(maxSize, maxFileAmount, config.AppName())
}

func NewDefaultMetricLogWriterOfApp(maxSize uint64, maxFileAmount uint32, appName string) (MetricLogWriter, error) {
	if maxSize == 0 || maxFileAmount == 0 {
		return nil, errors.New("invalid maxSize or maxFileAmount")
	}
	_, offset := time.Now().Zone()

	logDir := config.LogBaseDir()
	if len(logDir) == 0 {
		logDir = config.GetDefaultLogDir()
	}
	baseDir := logDir
	baseFilename := FormMetricFileName(appName, config.LogUsePid())

	writer := &DefaultMetricLogWriter{
		maxSingleSize:     maxSize,
		maxFileAmount:     maxFileAmount,
		timezoneOffsetSec: int64(offset),
		latestOpSec:       0,
		baseDir:           baseDir,
		baseFilename:      baseFilename,
		mux:               new(sync.RWMutex),
	}
	err := writer.initialize()
	return writer, err
}
