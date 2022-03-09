package holmes

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"
)

// Holmes is a self-aware profile dumper.
type Holmes struct {
	opts *options

	// stats
	changelog                int32
	collectCount             int
	gcCycleCount             int
	threadTriggerCount       int
	cpuTriggerCount          int
	memTriggerCount          int
	grTriggerCount           int
	gcHeapTriggerCount       int
	shrinkThreadTriggerCount int

	// channel for GC sweep finalizer event
	finCh chan struct{}

	// cooldown
	threadCoolDownTime    time.Time
	cpuCoolDownTime       time.Time
	memCoolDownTime       time.Time
	gcHeapCoolDownTime    time.Time
	grCoolDownTime        time.Time
	shrinkThrCoolDownTime time.Time

	// GC heap triggered, need to dump next time.
	gcHeapTriggered bool

	// stats ring
	memStats    ring
	cpuStats    ring
	grNumStats  ring
	threadStats ring
	gcHeapStats ring

	// switch
	stopped int64

	// profiler reporter channels
	rptEventsCh chan rptEvent
	rptCancelCh chan struct{}
}

type ProfileReporter interface {
	Report(pType string, buf []byte, reason string, eventID string) error
}

// New creates a holmes dumper.
func New(opts ...Option) (*Holmes, error) {
	holmes := &Holmes{
		opts:        newOptions(),
		finCh:       make(chan struct{}, 1),
		stopped:     1, // Initialization should be off
		rptEventsCh: make(chan rptEvent, 32),
		rptCancelCh: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		if err := opt.apply(holmes.opts); err != nil {
			return nil, err
		}
	}

	return holmes, nil
}

// EnableThreadDump enables the goroutine dump.
func (h *Holmes) EnableThreadDump() *Holmes {
	h.opts.threadOpts.Enable = true
	return h
}

// DisableThreadDump disables the goroutine dump.
func (h *Holmes) DisableThreadDump() *Holmes {
	h.opts.threadOpts.Enable = false
	return h
}

// EnableGoroutineDump enables the goroutine dump.
func (h *Holmes) EnableGoroutineDump() *Holmes {
	h.opts.grOpts.Enable = true
	return h
}

// DisableGoroutineDump disables the goroutine dump.
func (h *Holmes) DisableGoroutineDump() *Holmes {
	h.opts.grOpts.Enable = false
	return h
}

// EnableCPUDump enables the CPU dump.
func (h *Holmes) EnableCPUDump() *Holmes {
	h.opts.cpuOpts.Enable = true
	return h
}

// DisableCPUDump disables the CPU dump.
func (h *Holmes) DisableCPUDump() *Holmes {
	h.opts.cpuOpts.Enable = false
	return h
}

// EnableMemDump enables the mem dump.
func (h *Holmes) EnableMemDump() *Holmes {
	h.opts.memOpts.Enable = true
	return h
}

// DisableMemDump disables the mem dump.
func (h *Holmes) DisableMemDump() *Holmes {
	h.opts.gCHeapOpts.Enable = false
	return h
}

// EnableGCHeapDump enables the GC heap dump.
func (h *Holmes) EnableGCHeapDump() *Holmes {
	h.opts.gCHeapOpts.Enable = true
	return h
}

// DisableGCHeapDump disables the gc heap dump.
func (h *Holmes) DisableGCHeapDump() *Holmes {
	h.opts.gCHeapOpts.Enable = false
	return h
}

func finalizerCallback(gc *gcHeapFinalizer) {
	// disable or stop gc clean up normally
	if atomic.LoadInt64(&gc.h.stopped) == 1 {
		close(gc.h.finCh)
		return
	}

	// register the finalizer again
	runtime.SetFinalizer(gc, finalizerCallback)

	select {
	case gc.h.finCh <- struct{}{}:
	default:
		gc.h.logf("can not send event to finalizer channel immediately, may be analyzer blocked?")
	}
}

// it won't fit into tiny span since this struct contains point.
type gcHeapFinalizer struct {
	h *Holmes
}

func (h *Holmes) startGCCycleLoop() {
	h.gcHeapStats = newRing(minCollectCyclesBeforeDumpStart)

	gc := &gcHeapFinalizer{
		h,
	}

	runtime.SetFinalizer(gc, finalizerCallback)

	go gc.h.gcHeapCheckLoop()
}

// Start starts the dump loop of holmes.
func (h *Holmes) Start() {
	if !atomic.CompareAndSwapInt64(&h.stopped, 1, 0) {
		//nolint
		fmt.Println("Holmes has started, please don't start it again.")
		return
	}
	h.initEnvironment()
	go h.startDumpLoop()
	go h.startReporter()

	h.startGCCycleLoop()
}

// Stop the dump loop.
func (h *Holmes) Stop() {
	atomic.StoreInt64(&h.stopped, 1)
	h.stopReporter()
}

func (h *Holmes) startDumpLoop() {
	// init previous cool down time
	now := time.Now()
	h.cpuCoolDownTime = now
	h.memCoolDownTime = now
	h.grCoolDownTime = now

	// init stats ring
	h.cpuStats = newRing(minCollectCyclesBeforeDumpStart)
	h.memStats = newRing(minCollectCyclesBeforeDumpStart)
	h.grNumStats = newRing(minCollectCyclesBeforeDumpStart)
	h.threadStats = newRing(minCollectCyclesBeforeDumpStart)

	// dump loop
	ticker := time.NewTicker(h.opts.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.opts.intervalResetting:
			// wait for go version update to 1.15
			// can use Reset API directly here. pkg.go.dev/time#Ticker.Reset
			// we can't use the `for-range` here, because the range loop
			// caches the variable to be lopped and then it can't be overwritten
			itv := h.opts.CollectInterval
			fmt.Printf("[Holmes] collect interval is resetting to [%v]\n", itv) //nolint:forbidigo
			ticker = time.NewTicker(itv)

		default:
			// bug fix: https://github.com/mosn/holmes/issues/63
			// make sure that the message inside intervalResetting channel
			// would be consumed before ticker.C.
			<-ticker.C
			if atomic.LoadInt64(&h.stopped) == 1 {
				fmt.Println("[Holmes] dump loop stopped") //nolint:forbidigo
				return
			}

			cpuCore, err := h.getCPUCore()
			if cpuCore == 0 || err != nil {
				h.logf("[Holmes] get CPU core failed, CPU core: %v, error: %v", cpuCore, err)
				return
			}

			memoryLimit, err := h.getMemoryLimit()
			if memoryLimit == 0 || err != nil {
				h.logf("[Holmes] get memory limit failed, memory limit: %v, error: %v", memoryLimit, err)
				return
			}

			cpu, mem, gNum, tNum, err := collect(cpuCore, memoryLimit)
			if err != nil {
				h.logf(err.Error())

				continue
			}

			h.cpuStats.push(cpu)
			h.memStats.push(mem)
			h.grNumStats.push(gNum)
			h.threadStats.push(tNum)

			h.collectCount++
			if h.collectCount < minCollectCyclesBeforeDumpStart {
				// at least collect some cycles
				// before start to judge and dump
				h.logf("[Holmes] warming up cycle : %d", h.collectCount)

				continue
			}

			if err := h.EnableDump(cpu); err != nil {
				h.logf("[Holmes] unable to dump: %v", err)

				continue
			}

			h.goroutineCheckAndDump(gNum)
			h.memCheckAndDump(mem)
			h.cpuCheckAndDump(cpu)
			h.threadCheckAndDump(tNum)
			h.threadCheckAndShrink(tNum)
		}
	}
}

// goroutine start.
func (h *Holmes) goroutineCheckAndDump(gNum int) {
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown
	grOpts := h.opts.GetGrOpts()

	if !grOpts.Enable {
		return
	}

	if h.grCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] goroutine dump is in cooldown")
		return
	}
	// grOpts is a struct, no escape.
	if triggered := h.goroutineProfile(gNum, grOpts); triggered {
		h.grCoolDownTime = time.Now().Add(coolDown)
		h.grTriggerCount++
	}
}

func (h *Holmes) goroutineProfile(gNum int, c grOptions) bool {
	pType := type2name[goroutine]
	match, reason := matchRule(h.grNumStats, gNum, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, c.GoroutineTriggerNumMax)
	if !match {
		h.debugf(UniformLogFormat, "NODUMP", type2name[goroutine],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
			c.GoroutineTriggerNumMax, h.grNumStats.data, gNum)
		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeGrProfileDataToFile(buf, c, goroutine, gNum)

	h.ReportProfile(pType, buf.Bytes(), reason, "")
	return true
}

// memory start.
func (h *Holmes) memCheckAndDump(mem int) {
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown
	memOpts := h.opts.GetMemOpts()

	if !memOpts.Enable {
		return
	}

	if h.memCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] mem dump is in cooldown")
		return
	}
	// memOpts is a struct, no escape.
	if triggered := h.memProfile(mem, memOpts); triggered {
		h.memCoolDownTime = time.Now().Add(coolDown)
		h.memTriggerCount++
	}
}

func (h *Holmes) memProfile(rss int, c typeOption) bool {
	pType := type2name[mem]
	match, reason := matchRule(h.memStats, rss, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !match {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", pType,
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
			h.memStats.data, rss)

		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("heap").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck

	h.writeProfileDataToFile(buf, c, mem, rss, h.memStats, "")

	h.ReportProfile(pType, buf.Bytes(), reason, "")
	return true
}

func (h *Holmes) threadCheckAndShrink(threadNum int) {
	opts := h.opts.GetShrinkThreadOpts()

	if !opts.Enable {
		return
	}

	if h.shrinkThrCoolDownTime.After(time.Now()) {
		return
	}

	if threadNum > opts.Threshold {
		// 100x Delay time a cooldown time
		h.shrinkThrCoolDownTime = time.Now().Add(opts.Delay * 100)

		h.logf("current thread number(%v) larger than threshold(%v), will start to shrink thread after %v", threadNum, opts.Threshold, opts.Delay)
		time.AfterFunc(opts.Delay, func() {
			h.startShrinkThread()
		})
	}
}

// thread start.
func (h *Holmes) threadCheckAndDump(threadNum int) {
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown

	threadOpts := h.opts.GetThreadOpts()

	if !threadOpts.Enable {
		return
	}

	if h.threadCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] thread dump is in cooldown")
		return
	}
	// threadOpts is a struct, no escape.
	if triggered := h.threadProfile(threadNum, threadOpts); triggered {
		h.threadCoolDownTime = time.Now().Add(coolDown)
		h.threadTriggerCount++
	}
}

// TODO: better only shrink the threads that are idle.
func (h *Holmes) startShrinkThread() {

	curThreadNum := getThreadNum()
	opts := h.opts.GetShrinkThreadOpts()

	n := curThreadNum - opts.Threshold

	// check again after the timer triggered
	if opts.Enable && n > 0 {
		h.shrinkThreadTriggerCount++
		h.logf("start to shrink %v threads, now: %v", n, curThreadNum)

		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			// avoid close too much thread in batch.
			time.Sleep(time.Millisecond * 100)

			go func() {
				defer wg.Done()
				runtime.LockOSThread()
			}()
		}
		wg.Wait()

		h.logf("finished shrink threads, now: %v", getThreadNum())
	}
}

func (h *Holmes) threadProfile(curThreadNum int, c typeOption) bool {
	pType := type2name[thread]
	match, reason := matchRule(h.threadStats, curThreadNum, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !match {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", pType,
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
			h.threadStats.data, curThreadNum)

		return false
	}

	eventID := fmt.Sprintf("thr-%d", h.threadTriggerCount)

	var buf bytes.Buffer

	_ = pprof.Lookup("threadcreate").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, c, thread, curThreadNum, h.threadStats, eventID)

	h.ReportProfile(pType, buf.Bytes(), reason, eventID)

	buf.Reset()
	_ = pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, c, goroutine, curThreadNum, h.threadStats, eventID)

	h.ReportProfile("goroutine", buf.Bytes(), reason, eventID)

	return true
}

// thread end.

// cpu start.
func (h *Holmes) cpuCheckAndDump(cpu int) {
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown

	cpuOpts := h.opts.GetCPUOpts()

	if !cpuOpts.Enable {
		return
	}

	if h.cpuCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] cpu dump is in cooldown")
		return
	}
	// cpuOpts is a struct, no escape.
	if triggered := h.cpuProfile(cpu, cpuOpts); triggered {
		h.cpuCoolDownTime = time.Now().Add(coolDown)
		h.cpuTriggerCount++
	}
}

func (h *Holmes) cpuProfile(curCPUUsage int, c typeOption) bool {
	pType := type2name[cpu]
	match, reason := matchRule(h.cpuStats, curCPUUsage, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !match {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", pType,
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
			h.cpuStats.data, curCPUUsage)

		return false
	}

	binFileName := getBinaryFileName(h.opts.DumpPath, cpu, "")

	bf, err := os.OpenFile(binFileName, defaultLoggerFlags, defaultLoggerPerm)
	if err != nil {
		h.logf("[Holmes] failed to create cpu profile file: %v", err.Error())
		return false
	}
	defer bf.Close() // nolint: errcheck

	err = pprof.StartCPUProfile(bf)
	if err != nil {
		h.logf("[Holmes] failed to profile cpu: %v", err.Error())
		return false
	}

	time.Sleep(defaultCPUSamplingTime)
	pprof.StopCPUProfile()

	h.logf(UniformLogFormat, "pprof dump to log dir", type2name[cpu],
		c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
		h.cpuStats.data, curCPUUsage)

	if opts := h.opts.GetReporterOpts(); opts.active == 1 {
		bfCpy, err := ioutil.ReadFile(binFileName)
		if err != nil {
			h.logf("fail to build copy of bf, err %v", err)
			return true
		}
		h.ReportProfile(pType, bfCpy, reason, "")
	}

	return true
}

func (h *Holmes) gcHeapCheckLoop() {
	for {
		// wait for the finalizer event
		_, ok := <-h.finCh
		if !ok {
			// close finch?
			return
		}

		h.gcHeapCheckAndDump()
	}
}

func (h *Holmes) gcHeapCheckAndDump() {
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown

	gcHeapOpts := h.opts.GetGcHeapOpts()

	if !gcHeapOpts.Enable || atomic.LoadInt64(&h.stopped) == 1 {
		return
	}

	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)

	// TODO: we can only use NextGC for now since runtime haven't expose heapmarked yet
	// and we hard code the gcPercent is 100 here.
	// may introduce a new API debug.GCHeapMarked? it can also has better performance(no STW).
	nextGC := memStats.NextGC
	prevGC := nextGC / 2 //nolint:gomnd

	memoryLimit, err := h.getMemoryLimit()
	if memoryLimit == 0 || err != nil {
		h.logf("[Holmes] get memory limit failed, memory limit: %v, error: %v", memoryLimit, err)
		return
	}

	ratio := int(100 * float64(prevGC) / float64(memoryLimit))
	h.gcHeapStats.push(ratio)

	h.gcCycleCount++
	if h.gcCycleCount < minCollectCyclesBeforeDumpStart {
		// at least collect some cycles
		// before start to judge and dump
		h.logf("[Holmes] GC cycle warming up : %d", h.gcCycleCount)
		return
	}

	if h.gcHeapCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] GC heap dump is in cooldown")
		return
	}

	if triggered := h.gcHeapProfile(ratio, h.gcHeapTriggered, gcHeapOpts); triggered {
		if h.gcHeapTriggered {
			// already dump twice, mark it false
			h.gcHeapTriggered = false
			h.gcHeapCoolDownTime = time.Now().Add(coolDown)
			h.gcHeapTriggerCount++
		} else {
			// force dump next time
			h.gcHeapTriggered = true
		}
	}
}

func (h *Holmes) getCPUCore() (float64, error) {
	if h.opts.cpuCore > 0 {
		return h.opts.cpuCore, nil
	}

	if h.opts.UseGoProcAsCPUCore {
		return float64(runtime.GOMAXPROCS(-1)), nil
	}

	if h.opts.UseCGroup {
		return getCGroupCPUCore()
	}

	return float64(runtime.NumCPU()), nil
}

func (h *Holmes) getMemoryLimit() (uint64, error) {
	if h.opts.memoryLimit > 0 {
		return h.opts.memoryLimit, nil
	}

	if h.opts.UseCGroup {
		return getCGroupMemoryLimit()
	}

	return getNormalMemoryLimit()
}

// gcHeapProfile will dump profile twice when triggered once.
// since the current memory profile will be merged after next GC cycle.
// And we assume the finalizer will be called before next GC cycle(it will be usually).
func (h *Holmes) gcHeapProfile(gc int, force bool, c typeOption) bool {
	pType := type2name[gcHeap]
	match, reason := matchRule(h.gcHeapStats, gc, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !force && !match {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", pType,
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
			NotSupportTypeMaxConfig,
			h.gcHeapStats.data, gc)

		return false
	}

	// gcTriggerCount only increased after got both two profiles
	eventID := fmt.Sprintf("heap-%d", h.grTriggerCount)

	var buf bytes.Buffer
	_ = pprof.Lookup("heap").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, c, gcHeap, gc, h.gcHeapStats, eventID)

	h.ReportProfile(pType, buf.Bytes(), reason, eventID)
	return true
}

func (h *Holmes) writeGrProfileDataToFile(data bytes.Buffer, opts grOptions, dumpType configureType, currentStat int) {
	h.logf(UniformLogFormat, "pprof", type2name[dumpType],
		opts.TriggerMin, opts.TriggerDiff, opts.TriggerAbs,
		opts.GoroutineTriggerNumMax,
		h.grNumStats.data, currentStat)

	if err := writeFile(data, dumpType, h.opts.DumpOptions, ""); err != nil {
		h.logf("%s", err.Error())
	}
}

func (h *Holmes) writeProfileDataToFile(data bytes.Buffer, opts typeOption, dumpType configureType, currentStat int, ringStats ring, eventID string) {
	h.logf(UniformLogFormat, "pprof", type2name[dumpType],
		opts.TriggerMin, opts.TriggerDiff, opts.TriggerAbs,
		NotSupportTypeMaxConfig, ringStats, currentStat)

	if err := writeFile(data, dumpType, h.opts.DumpOptions, eventID); err != nil {
		h.logf("%s", err.Error())
	}
}

func (h *Holmes) initEnvironment() {
	// whether the max memory is limited by cgroup
	if h.opts.UseCGroup {
		h.logf("[Holmes] use cgroup to limit memory")
	} else {
		h.logf("[Holmes] use the default memory percent calculated by gopsutil")
	}

	logger := h.opts.Logger.Load()

	if (logger == nil || logger == os.Stdout) && h.opts.logOpts.RotateEnable {
		h.opts.logOpts.RotateEnable = false
	}
}

func (h *Holmes) EnableDump(curCPU int) (err error) {
	if h.opts.CPUMaxPercent != 0 && curCPU >= h.opts.CPUMaxPercent {
		return fmt.Errorf("current cpu percent [%v] is greater than the CPUMaxPercent [%v]", cpu, h.opts.CPUMaxPercent)
	}
	return nil
}

// Set sets holmes's optional after initialing.
func (h *Holmes) Set(opts ...Option) error {
	h.opts.L.Lock()
	defer h.opts.L.Unlock()

	for _, opt := range opts {
		if err := opt.apply(h.opts); err != nil {
			return err
		}
	}
	return nil
}

func (h *Holmes) DisableProfileReporter() {
	atomic.StoreInt32(&h.opts.rptOpts.active, 0)
}

func (h *Holmes) ReportProfile(pType string, buf []byte, reason string, eventID string) {
	opts := h.opts.GetReporterOpts()
	if opts.active == 0 {
		return
	}
	h.rptEventsCh <- rptEvent{
		pType,
		buf,
		reason,
		eventID}
}

// startReporter starts a background goroutine to consume event channel,
// and finish it at after receive from cancel channel.
func (h *Holmes) startReporter() {
	go func() {
		for {
			select {
			case <-h.rptCancelCh:
				h.logf("stop reporter background goroutine")
				return
			default:
				evt := <-h.rptEventsCh
				opts := h.opts.GetReporterOpts()
				if opts.reporter == nil {
					h.logf("reporter is nil, please initial it before startReporter")
					// drop the event
					continue
				}

				if opts.active == 0 {
					//drop the event
					continue
				}
				opts.reporter.Report(evt.PType, evt.Buf, evt.Reason, evt.EventID) // nolint: errcheck
			}
		}
	}()
}

func (h *Holmes) stopReporter() {
	h.rptCancelCh <- struct{}{}
}
