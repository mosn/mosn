package metric

import (
	"bufio"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/pkg/errors"
	"io"
	"os"
)

const maxItemAmount = 100000

type MetricLogReader interface {
	ReadMetrics(nameList []string, fileNo uint32, startOffset uint64, maxLines uint32) ([]*base.MetricItem, error)

	ReadMetricsByEndTime(nameList []string, fileNo uint32, startOffset uint64, beginMs uint64, endMs uint64, resource string) ([]*base.MetricItem, error)
}

// Not thread-safe itself, but guarded by the outside MetricSearcher.
type defaultMetricLogReader struct {
}

func (r *defaultMetricLogReader) ReadMetrics(nameList []string, fileNo uint32, startOffset uint64, maxLines uint32) ([]*base.MetricItem, error) {
	if len(nameList) == 0 {
		return make([]*base.MetricItem, 0), nil
	}
	// startOffset: the offset of the first file to read
	items, shouldContinue, err := r.readMetricsInOneFile(nameList[fileNo], startOffset, maxLines, 0, 0)
	if err != nil {
		return nil, err
	}
	if !shouldContinue {
		return items, nil
	}
	fileNo++
	// Continue reading until the size or time does not satisfy the condition
	for {
		if int(fileNo) >= len(nameList) || len(items) >= int(maxLines) {
			// No files to read.
			break
		}
		arr, shouldContinue, err := r.readMetricsInOneFile(nameList[fileNo], 0, maxLines, getLatestSecond(items), uint32(len(items)))
		if err != nil {
			return nil, err
		}
		items = append(items, arr...)
		if !shouldContinue {
			break
		}
		fileNo++
	}
	return items, nil
}

func (r *defaultMetricLogReader) ReadMetricsByEndTime(nameList []string, fileNo uint32, startOffset uint64, beginMs uint64, endMs uint64, resource string) ([]*base.MetricItem, error) {
	if len(nameList) == 0 {
		return make([]*base.MetricItem, 0), nil
	}
	// startOffset: the offset of the first file to read
	items, shouldContinue, err := r.readMetricsInOneFileByEndTime(nameList[fileNo], startOffset, beginMs, endMs, resource, 0)
	if err != nil {
		return nil, err
	}
	if !shouldContinue {
		return items, nil
	}
	fileNo++
	// Continue reading until the size or time does not satisfy the condition
	for {
		if int(fileNo) >= len(nameList) {
			// No files to read.
			break
		}
		arr, shouldContinue, err := r.readMetricsInOneFileByEndTime(nameList[fileNo], 0, beginMs, endMs, resource, uint32(len(items)))
		if err != nil {
			return nil, err
		}
		items = append(items, arr...)
		if !shouldContinue {
			break
		}
		fileNo++
	}
	return items, nil
}

func (r *defaultMetricLogReader) readMetricsInOneFile(filename string, offset uint64, maxLines uint32, lastSec uint64, prevSize uint32) ([]*base.MetricItem, bool, error) {
	file, err := openFileAndSeekTo(filename, offset)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	bufReader := bufio.NewReaderSize(file, 8192)
	items := make([]*base.MetricItem, 0)
	for {
		line, err := readLine(bufReader)
		if err != nil {
			if err == io.EOF {
				shouldContinue := prevSize+uint32(len(items)) < maxLines
				return items, shouldContinue, nil
			}
			return nil, false, errors.Wrap(err, "error when reading lines from file")
		}
		item, err := base.MetricItemFromFatString(line)
		if err != nil {
			logger.Errorf("Failed to convert MetricItem to string: %+v", err)
			continue
		}
		tsSec := item.Timestamp / 1000

		if prevSize+uint32(len(items)) >= maxLines && tsSec != lastSec {
			return items, false, nil
		}
		items = append(items, item)
		lastSec = tsSec
	}
}

func (r *defaultMetricLogReader) readMetricsInOneFileByEndTime(filename string, offset uint64, beginMs uint64, endMs uint64, resource string, prevSize uint32) ([]*base.MetricItem, bool, error) {
	beginSec := beginMs / 1000
	endSec := endMs / 1000
	file, err := openFileAndSeekTo(filename, offset)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	bufReader := bufio.NewReaderSize(file, 8192)
	items := make([]*base.MetricItem, 0)
	for {
		line, err := readLine(bufReader)
		if err != nil {
			if err == io.EOF {
				return items, true, nil
			}
			return nil, false, errors.Wrap(err, "error when reading lines from file")
		}
		item, err := base.MetricItemFromFatString(line)
		if err != nil {
			logger.Errorf("Invalid line of metric file: %s, error: %+v", line, err)
			continue
		}
		tsSec := item.Timestamp / 1000
		// currentSecond should in [beginSec, endSec]
		if tsSec < beginSec || tsSec > endSec {
			return items, false, nil
		}

		// empty resource name indicates "fetch all"
		if resource == "" || resource == item.Resource {
			items = append(items, item)
		}
		// Max items limit to avoid infinite reading
		if len(items)+int(prevSize) >= maxItemAmount {
			return items, false, nil
		}
	}
}

func readLine(bufReader *bufio.Reader) (string, error) {
	buf := make([]byte, 0)
	for {
		line, ne, err := bufReader.ReadLine()
		if err != nil {
			return "", err
		}
		buf = append(buf, line...)
		if !ne {
			return string(buf), err
		}
		// buffer size < line size, so we need to read until the `ne` flag is false.
	}
}

func getLatestSecond(items []*base.MetricItem) uint64 {
	if items == nil {
		return 0
	}
	return items[len(items)-1].Timestamp / 1000
}

func openFileAndSeekTo(filename string, offset uint64) (*os.File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file: "+filename)
	}

	// Set position to the offset recorded in the idx file
	_, err = file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		_ = file.Close()
		return nil, errors.Wrapf(err, "failed to fseek to offset %d", offset)
	}
	return file, nil
}

func newDefaultMetricLogReader() MetricLogReader {
	return &defaultMetricLogReader{}
}
