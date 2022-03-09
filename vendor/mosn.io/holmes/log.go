package holmes

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// log write content to log file.
func (h *Holmes) logf(pattern string, args ...interface{}) {
	if h.opts.LogLevel >= LogLevelInfo {
		timestamp := "[" + time.Now().Format("2006-01-02 15:04:05.000") + "]"
		h.writeString(fmt.Sprintf(timestamp+pattern+"\n", args...))
	}
}

// log write content to log file.
func (h *Holmes) debugf(pattern string, args ...interface{}) {
	if h.opts.LogLevel >= LogLevelDebug {
		h.writeString(fmt.Sprintf(pattern+"\n", args...))
	}
}

func (h *Holmes) writeString(content string) {
	logger, ok := h.opts.Logger.Load().(*os.File)
	if !ok || logger == nil {
		//nolint
		fmt.Println("write fail,logger is null or assert fail ", content) // where to write this log?
		return
	}

	if _, err := h.opts.Logger.Load().(*os.File).WriteString(content); err != nil {
		//nolint
		fmt.Println(err) // where to write this log?
		return
	}

	if !h.opts.logOpts.RotateEnable {
		return
	}

	state, err := logger.Stat()
	if err != nil {
		h.opts.logOpts.RotateEnable = false
		//nolint
		fmt.Println("get logger stat:", err, "from now on, it will be disabled split log")

		return
	}

	if state.Size() > h.opts.logOpts.SplitLoggerSize && atomic.CompareAndSwapInt32(&h.changelog, 0, 1) {
		defer atomic.StoreInt32(&h.changelog, 0)

		var (
			newLogger *os.File
			err       error
			dumpPath  = h.opts.DumpPath
			suffix    = time.Now().Format("20060102150405")
			srcPath   = filepath.Clean(filepath.Join(dumpPath, defaultLoggerName))
			dstPath   = srcPath + "_" + suffix + ".back"
		)

		err = os.Rename(srcPath, dstPath)

		if err != nil {
			h.opts.logOpts.RotateEnable = false
			//nolint
			fmt.Println("rename err:", err, "from now on, it will be disabled split log")

			return
		}

		newLogger, err = os.OpenFile(filepath.Clean(srcPath), defaultLoggerFlags, defaultLoggerPerm)

		if err != nil {
			h.opts.logOpts.RotateEnable = false

			//nolint
			fmt.Println("open new file err:", err, "from now on, it will be disabled split log")

			return
		}

		old := logger

		h.opts.Logger.Store(newLogger)

		_ = old.Close()
	}
}
