package holmes

import (
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/go-units"
)

type options struct {
	UseGoProcAsCPUCore bool // use the go max procs number as the CPU core number when it's true
	UseCGroup          bool // use the CGroup to calc cpu/memory when it's true

	// overwrite the system level memory limitation when > 0.
	memoryLimit uint64
	cpuCore     float64

	*ShrinkThrOptions

	*DumpOptions

	LogLevel int
	// Logger *os.File
	Logger atomic.Value

	// interval for dump loop, default 5s
	CollectInterval   time.Duration
	intervalResetting chan struct{}

	// the cooldown time after every type of dump
	// interval for cooldown，default 1m
	// the cpu/mem/goroutine have different cooldowns of their own

	// todo should we move CoolDown into Gr/CPU/MEM/GCheap Opts and support
	// set different `CoolDown` for different opts?
	CoolDown time.Duration

	// if current cpu usage percent is greater than CPUMaxPercent,
	// holmes would not dump all types profile, cuz this
	// move may result of the system crash.
	CPUMaxPercent int

	// if write lock is held mean holmes's
	// configuration is being modified.
	L *sync.RWMutex

	logOpts *loggerOptions
	grOpts  *grOptions

	memOpts    *typeOption
	gCHeapOpts *typeOption
	cpuOpts    *typeOption
	threadOpts *typeOption

	// profile reporter
	rptOpts *ReporterOptions
}

// rptEvent stands of the args of report event
type rptEvent struct {
	PType   string
	Buf     []byte
	Reason  string
	EventID string
}

type ReporterOptions struct {
	reporter ProfileReporter
	active   int32 // switch
}

// newReporterOpts returns  ReporterOptions。
func newReporterOpts() *ReporterOptions {
	opts := &ReporterOptions{}

	return opts
}

// DumpOptions contains configuration about dump file.
type DumpOptions struct {
	// full path to put the profile files, default /tmp
	DumpPath string
	// default dump to binary profile, set to true if you want a text profile
	DumpProfileType dumpProfileType
	// only dump top 10 if set to false, otherwise dump all, only effective when in_text = true
	DumpFullStack bool
}

// ShrinkThrOptions contains the configuration about shrink thread
type ShrinkThrOptions struct {
	// shrink the thread number when it exceeds the max threshold that specified in Threshold
	Enable    bool
	Threshold int
	Delay     time.Duration // start to shrink thread after the delay time.
}

// GetReporterOpts returns a copy of rptOpts.
func (o *options) GetReporterOpts() ReporterOptions {
	o.L.RLock()
	defer o.L.RUnlock()
	return *o.rptOpts
}

// GetShrinkThreadOpts return a copy of ShrinkThrOptions.
func (o *options) GetShrinkThreadOpts() ShrinkThrOptions {
	o.L.RLock()
	defer o.L.RUnlock()
	return *o.ShrinkThrOptions
}

// GetMemOpts return a copy of typeOption.
func (o *options) GetMemOpts() typeOption {
	o.L.RLock()
	defer o.L.RUnlock()
	return *o.memOpts
}

// GetCPUOpts return a copy of typeOption
// if cpuOpts not exist return a empty typeOption and false.
func (o *options) GetCPUOpts() typeOption {
	o.L.RLock()
	defer o.L.RUnlock()
	return *o.cpuOpts
}

// GetGrOpts return a copy of grOptions
// if grOpts not exist return a empty grOptions and false.
func (o *options) GetGrOpts() grOptions {
	o.L.RLock()
	defer o.L.RUnlock()
	return *o.grOpts
}

// GetThreadOpts return a copy of typeOption
// if threadOpts not exist return a empty typeOption and false.
func (o *options) GetThreadOpts() typeOption {
	o.L.RLock()
	defer o.L.RUnlock()
	return *o.threadOpts
}

// GetGcHeapOpts return a copy of typeOption
// if gCHeapOpts not exist return a empty typeOption and false.
func (o *options) GetGcHeapOpts() typeOption {
	o.L.RLock()
	defer o.L.RUnlock()
	return *o.gCHeapOpts
}

// Option holmes option type.
type Option interface {
	apply(*options) error
}

type optionFunc func(*options) error

func (f optionFunc) apply(opts *options) error {
	return f(opts)
}

func newOptions() *options {
	o := &options{
		logOpts:           newLoggerOptions(),
		grOpts:            newGrOptions(),
		memOpts:           newMemOptions(),
		gCHeapOpts:        newGCHeapOptions(),
		cpuOpts:           newCPUOptions(),
		threadOpts:        newThreadOptions(),
		LogLevel:          LogLevelDebug,
		CollectInterval:   defaultInterval,
		intervalResetting: make(chan struct{}, 1),
		CoolDown:          defaultCooldown,
		DumpOptions: &DumpOptions{
			DumpPath:        defaultDumpPath,
			DumpProfileType: defaultDumpProfileType,
			DumpFullStack:   false,
		},
		ShrinkThrOptions: &ShrinkThrOptions{
			Enable: false,
		},
		L:       &sync.RWMutex{},
		rptOpts: newReporterOpts(),
	}
	o.Logger.Store(os.Stdout)
	return o
}

// WithCollectInterval : interval must be valid time duration string,
// eg. "ns", "us" (or "µs"), "ms", "s", "m", "h".
func WithCollectInterval(interval string) Option {
	return optionFunc(func(opts *options) (err error) {
		// CollectInterval wouldn't be zero value, because it
		// will be initialized as defaultInterval at newOptions()
		newInterval, err := time.ParseDuration(interval)
		if err != nil || opts.CollectInterval.Seconds() == newInterval.Seconds() {
			return
		}

		opts.CollectInterval = newInterval
		opts.intervalResetting <- struct{}{}

		return
	})
}

// WithCoolDown : coolDown must be valid time duration string,
// eg. "ns", "us" (or "µs"), "ms", "s", "m", "h".
func WithCoolDown(coolDown string) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CoolDown, err = time.ParseDuration(coolDown)
		return
	})
}

// WithCPUMax : set the CPUMaxPercent parameter as max
func WithCPUMax(max int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CPUMaxPercent = max
		return
	})
}

// WithDumpPath set the dump path for holmes.
func WithDumpPath(dumpPath string, loginfo ...string) Option {
	return optionFunc(func(opts *options) (err error) {
		var logger *os.File
		f := path.Join(dumpPath, defaultLoggerName)
		if len(loginfo) > 0 {
			f = dumpPath + "/" + path.Join(loginfo...)
		}
		opts.DumpPath = filepath.Dir(f)
		logger, err = os.OpenFile(filepath.Clean(f), defaultLoggerFlags, defaultLoggerPerm)
		if err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(opts.DumpPath, 0755); err != nil {
				return
			}
			logger, err = os.OpenFile(filepath.Clean(f), defaultLoggerFlags, defaultLoggerPerm)
			if err != nil {
				return
			}
		}
		old, ok := opts.Logger.Load().(*os.File)
		if ok && logger != nil {
			_ = old.Close()
		}
		opts.Logger.Store(logger)
		return
	})
}

// WithBinaryDump set dump mode to binary.
func WithBinaryDump() Option {
	return withDumpProfileType(binaryDump)
}

// WithTextDump set dump mode to text.
func WithTextDump() Option {
	return withDumpProfileType(textDump)
}

// WithFullStack set to dump full stack or top 10 stack, when dump in text mode.
func WithFullStack(isFull bool) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.DumpFullStack = isFull
		return
	})
}

func withDumpProfileType(profileType dumpProfileType) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.DumpProfileType = profileType
		return
	})
}

type grOptions struct {
	// enable the goroutine dumper, should dump if one of the following requirements is matched
	//   1. goroutine_num > TriggerMin && goroutine_num < GoroutineTriggerNumMax && goroutine diff percent > TriggerDiff
	//   2. goroutine_num > GoroutineTriggerNumAbsNum && goroutine_num < GoroutineTriggerNumMax
	*typeOption
	GoroutineTriggerNumMax int // goroutine trigger max in number
}

func newGrOptions() *grOptions {
	base := newTypeOpts(
		defaultGoroutineTriggerMin,
		defaultGoroutineTriggerAbs,
		defaultGoroutineTriggerDiff)
	return &grOptions{typeOption: base}
}

// WithGoroutineDump set the goroutine dump options.
func WithGoroutineDump(min int, diff int, abs int, max int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.grOpts.Set(min, abs, diff)
		opts.grOpts.GoroutineTriggerNumMax = max
		return
	})
}

type typeOption struct {
	Enable bool
	// mem/cpu/gcheap trigger minimum in percent, goroutine/thread trigger minimum in number
	TriggerMin int

	// mem/cpu/gcheap trigger abs in percent, goroutine/thread trigger abs in number
	TriggerAbs int

	// mem/cpu/gcheap/goroutine/thread trigger diff in percent
	TriggerDiff int
}

func newTypeOpts(triggerMin, triggerAbs, triggerDiff int) *typeOption {
	return &typeOption{
		Enable:      false,
		TriggerMin:  triggerMin,
		TriggerAbs:  triggerAbs,
		TriggerDiff: triggerDiff,
	}
}

func (base *typeOption) Set(min, abs, diff int) {
	base.TriggerMin, base.TriggerAbs, base.TriggerDiff = min, abs, diff
}

// newMemOptions
// enable the heap dumper, should dump if one of the following requirements is matched
//   1. memory usage > TriggerMin && memory usage diff > TriggerDiff
//   2. memory usage > TriggerAbs.
func newMemOptions() *typeOption {
	return newTypeOpts(
		defaultMemTriggerMin,
		defaultMemTriggerAbs,
		defaultMemTriggerDiff)
}

// WithMemDump set the memory dump options.
func WithMemDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.memOpts.Set(min, abs, diff)
		return
	})
}

// newGCHeapOptions
// enable the heap dumper, should dump if one of the following requirements is matched
//   1. GC heap usage > TriggerMin && GC heap usage diff > TriggerDiff
//   2. GC heap usage > TriggerAbs
// in percent.
func newGCHeapOptions() *typeOption {
	return newTypeOpts(
		defaultGCHeapTriggerMin,
		defaultGCHeapTriggerAbs,
		defaultGCHeapTriggerDiff)
}

// WithGCHeapDump set the GC heap dump options.
func WithGCHeapDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.gCHeapOpts.Set(min, abs, diff)
		return
	})
}

// WithCPUCore overwrite the system level CPU core number when it > 0.
// it's not a good idea to modify it on fly since it affects the CPU percent caculation.
func WithCPUCore(cpuCore float64) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.cpuCore = cpuCore
		return
	})
}

// WithMemoryLimit overwrite the system level memory limit when it > 0.
func WithMemoryLimit(limit uint64) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.memoryLimit = limit
		return
	})
}

func newThreadOptions() *typeOption {
	return newTypeOpts(
		defaultThreadTriggerMin,
		defaultThreadTriggerAbs,
		defaultThreadTriggerDiff)
}

// WithThreadDump set the thread dump options.
func WithThreadDump(min, diff, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.threadOpts.Set(min, abs, diff)
		return
	})
}

// newCPUOptions
// enable the cpu dumper, should dump if one of the following requirements is matched
// in percent
//   1. cpu usage > CPUTriggerMin && cpu usage diff > CPUTriggerDiff
//   2. cpu usage > CPUTriggerAbs
// in percent.
func newCPUOptions() *typeOption {
	return newTypeOpts(
		defaultCPUTriggerMin,
		defaultCPUTriggerAbs,
		defaultCPUTriggerDiff)
}

// WithCPUDump set the cpu dump options.
func WithCPUDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.cpuOpts.Set(min, abs, diff)
		return
	})
}

// WithGoProcAsCPUCore set holmes use cgroup or not.
func WithGoProcAsCPUCore(enabled bool) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.UseGoProcAsCPUCore = enabled
		return
	})
}

// WithCGroup set holmes use cgroup or not.
func WithCGroup(useCGroup bool) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.UseCGroup = useCGroup
		return
	})
}

func WithLoggerLevel(level int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.LogLevel = level
		return
	})
}

type loggerOptions struct {
	RotateEnable    bool
	SplitLoggerSize int64 // SplitLoggerSize The size of the log split
}

func newLoggerOptions() *loggerOptions {
	return &loggerOptions{
		RotateEnable:    true,
		SplitLoggerSize: defaultShardLoggerSize,
	}
}

// WithLoggerSplit set the split log options.
// eg. "b/B", "k/K" "kb/Kb" "mb/Mb", "gb/Gb" "tb/Tb" "pb/Pb".
func WithLoggerSplit(enable bool, shardLoggerSize string) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.logOpts.RotateEnable = enable
		if !enable {
			return nil
		}

		parseShardLoggerSize, err := units.FromHumanSize(shardLoggerSize)
		if err != nil || (err == nil && parseShardLoggerSize <= 0) {
			opts.logOpts.SplitLoggerSize = defaultShardLoggerSize
			return
		}

		opts.logOpts.SplitLoggerSize = parseShardLoggerSize
		return
	})
}

// WithShrinkThread enable/disable shrink thread when the thread number exceed the max threshold.
func WithShrinkThread(enable bool, threshold int, delay time.Duration) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.ShrinkThrOptions.Enable = enable
		if threshold > 0 {
			opts.ShrinkThrOptions.Threshold = threshold
		}
		opts.ShrinkThrOptions.Delay = delay
		return
	})
}

// WithProfileReporter will enable reporter
// reopens profile reporter through WithProfileReporter(h.opts.rptOpts.reporter)
func WithProfileReporter(r ProfileReporter) Option {
	return optionFunc(func(opts *options) (err error) {
		if r == nil {
			return nil
		}

		opts.rptOpts.reporter = r
		atomic.StoreInt32(&opts.rptOpts.active, 1)
		return
	})
}
