package holmes

import (
	"os"
	"time"
)

const (
	defaultThreadTriggerMin  = 10 // 10 threads
	defaultThreadTriggerAbs  = 70 // 70 threads
	defaultThreadTriggerDiff = 25 // 25%

	defaultCPUTriggerMin   = 10              // 10%
	defaultCPUTriggerAbs   = 70              // 70%
	defaultCPUTriggerDiff  = 25              // 25%
	defaultCPUSamplingTime = 5 * time.Second // collect 5s cpu profile

	defaultGoroutineTriggerMin  = 3000   // 3000 goroutines
	defaultGoroutineTriggerAbs  = 200000 // 200k goroutines
	defaultGoroutineTriggerDiff = 20     // 20%  diff

	defaultMemTriggerMin  = 10 // 10%
	defaultMemTriggerAbs  = 80 // 80%
	defaultMemTriggerDiff = 25 // 25%

	defaultGCHeapTriggerMin  = 10 // 10%
	defaultGCHeapTriggerAbs  = 40 // 40%
	defaultGCHeapTriggerDiff = 20 // 20%

	defaultInterval        = 5 * time.Second
	defaultCooldown        = time.Minute
	defaultDumpProfileType = binaryDump
	defaultDumpPath        = "/tmp"
	defaultLoggerName      = "holmes.log"
	defaultLoggerFlags     = os.O_RDWR | os.O_CREATE | os.O_APPEND
	defaultLoggerPerm      = 0644
	defaultShardLoggerSize = 5242880 // 5m
)

type dumpProfileType int

const (
	binaryDump dumpProfileType = 0
	textDump   dumpProfileType = 1
)

type configureType int

const (
	mem configureType = iota
	cpu
	thread
	goroutine
	gcHeap
)

var type2name = map[configureType]string{
	mem:       "mem",
	cpu:       "cpu",
	thread:    "thread",
	goroutine: "goroutine",
	gcHeap:    "GCHeap",
}

const (
	cgroupMemLimitPath  = "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	cgroupCpuQuotaPath  = "/sys/fs/cgroup/cpu/cpu.cfs_quota_us"
	cgroupCpuPeriodPath = "/sys/fs/cgroup/cpu/cpu.cfs_period_us"
)

const minCollectCyclesBeforeDumpStart = 10

type Lever int8

const (
	LogLevelDebug Lever = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var m = map[Lever]string{
	LogLevelDebug: "[Debug]",
	LogLevelInfo:  "[Info]",
	LogLevelWarn:  "[Warn]",
	LogLevelError: "[Error]",
}

// Allow 允许是否可以打印
func (l Lever) Allow(lv Lever) bool {
	return lv >= l
}

// String 语义转义
func (l Lever) String() string {
	if v, ok := m[l]; ok {
		return v
	}
	return "UNKNOWN"
}

const (
	// TrimResultTopN trimResult return only reserve the top n.
	TrimResultTopN = 10

	// NotSupportTypeMaxConfig means this profile type is
	// not support control dump profile by max parameter.
	NotSupportTypeMaxConfig = 0

	// UniformLogFormat is the format of uniform logging.
	UniformLogFormat = "[Holmes] %v %v, config_min : %v, config_diff : %v, config_abs : %v, config_max : %v, previous : %v, current: %v"
)
