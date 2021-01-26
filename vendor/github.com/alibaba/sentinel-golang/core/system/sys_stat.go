package system

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alibaba/sentinel-golang/util"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
)

const (
	notRetrievedValue float64 = -1
)

var (
	currentLoad     atomic.Value
	currentCpuUsage atomic.Value

	prevCpuStat *cpu.TimesStat
	initOnce    sync.Once

	ssStopChan = make(chan struct{})
)

func init() {
	currentLoad.Store(notRetrievedValue)
	currentCpuUsage.Store(notRetrievedValue)
}

func InitCollector(intervalMs uint32) {
	if intervalMs == 0 {
		return
	}
	initOnce.Do(func() {
		// Initial retrieval.
		retrieveAndUpdateSystemStat()

		ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		go util.RunWithRecover(func() {
			for {
				select {
				case <-ticker.C:
					retrieveAndUpdateSystemStat()
				case <-ssStopChan:
					ticker.Stop()
					return
				}
			}
		}, logger)
	})
}

func retrieveAndUpdateSystemStat() {
	cpuStats, err := cpu.Times(false)
	if err != nil {
		logger.Warnf("Failed to retrieve current CPU usage: %+v", err)
	}
	loadStat, err := load.Avg()
	if err != nil {
		logger.Warnf("Failed to retrieve current system load: %+v", err)
	}
	if len(cpuStats) > 0 {
		curCpuStat := &cpuStats[0]
		recordCpuUsage(prevCpuStat, curCpuStat)
		// Cache the latest CPU stat info.
		prevCpuStat = curCpuStat
	}
	if loadStat != nil {
		currentLoad.Store(loadStat.Load1)
	}
}

func recordCpuUsage(prev, curCpuStat *cpu.TimesStat) {
	if prev != nil && curCpuStat != nil {
		prevTotal := calculateTotalCpuTick(prev)
		curTotal := calculateTotalCpuTick(curCpuStat)

		tDiff := curTotal - prevTotal
		var cpuUsage float64
		if tDiff == 0 {
			cpuUsage = 0
		} else {
			prevUsed := calculateUserCpuTick(prev) + calculateKernelCpuTick(prev)
			curUsed := calculateUserCpuTick(curCpuStat) + calculateKernelCpuTick(curCpuStat)
			cpuUsage = (curUsed - prevUsed) / tDiff
			cpuUsage = math.Max(0.0, cpuUsage)
			cpuUsage = math.Min(1.0, cpuUsage)
		}
		currentCpuUsage.Store(cpuUsage)
	}
}

func calculateTotalCpuTick(stat *cpu.TimesStat) float64 {
	return stat.User + stat.Nice + stat.System + stat.Idle +
		stat.Iowait + stat.Irq + stat.Softirq + stat.Steal
}

func calculateUserCpuTick(stat *cpu.TimesStat) float64 {
	return stat.User + stat.Nice
}

func calculateKernelCpuTick(stat *cpu.TimesStat) float64 {
	return stat.System + stat.Irq + stat.Softirq
}

func CurrentLoad() float64 {
	r, ok := currentLoad.Load().(float64)
	if !ok {
		return notRetrievedValue
	}
	return r
}

func CurrentCpuUsage() float64 {
	r, ok := currentCpuUsage.Load().(float64)
	if !ok {
		return notRetrievedValue
	}
	return r
}
