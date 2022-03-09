package holmes

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	mem_util "github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

// copied from https://github.com/containerd/cgroups/blob/318312a373405e5e91134d8063d04d59768a1bff/utils.go#L251
func parseUint(s string, base, bitSize int) (uint64, error) {
	v, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil &&
			intErr.(*strconv.NumError).Err == strconv.ErrRange &&
			intValue < 0 {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// copied from https://github.com/containerd/cgroups/blob/318312a373405e5e91134d8063d04d59768a1bff/utils.go#L243
func readUint(path string) (uint64, error) {
	v, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseUint(strings.TrimSpace(string(v)), 10, 64)
}

// only reserve the top n.
func trimResult(buffer bytes.Buffer) string {
	index := TrimResultTopN
	arr := strings.SplitN(buffer.String(), "\n\n", TrimResultTopN+1)

	if len(arr) <= TrimResultTopN {
		index = len(arr) - 1
	}

	return strings.Join(arr[:index], "\n\n")
}

// return values:
// 1. cpu percent, not division cpu cores yet,
// 2. RSS mem in bytes,
// 3. goroutine num,
// 4. thread num
func getUsage() (float64, uint64, int, int, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, 0, 0, 0, err
	}
	cpuPercent, err := p.Percent(time.Second)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	mem, err := p.MemoryInfo()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	rss := mem.RSS
	gNum := runtime.NumGoroutine()
	tNum := getThreadNum()

	return cpuPercent, rss, gNum, tNum, nil
}

// get cpu core number limited by CGroup.
func getCGroupCPUCore() (float64, error) {
	var cpuQuota uint64

	cpuPeriod, err := readUint(cgroupCpuPeriodPath)
	if cpuPeriod == 0 || err != nil {
		return 0, err
	}

	if cpuQuota, err = readUint(cgroupCpuQuotaPath); err != nil {
		return 0, err
	}

	return float64(cpuQuota) / float64(cpuPeriod), nil
}

func getCGroupMemoryLimit() (uint64, error) {
	usage, err := readUint(cgroupMemLimitPath)
	if err != nil {
		return 0, err
	}
	machineMemory, err := mem_util.VirtualMemory()
	if err != nil {
		return 0, err
	}
	limit := uint64(math.Min(float64(usage), float64(machineMemory.Total)))
	return limit, nil
}

func getNormalMemoryLimit() (uint64, error) {
	machineMemory, err := mem_util.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return machineMemory.Total, nil
}

func getThreadNum() int {
	return pprof.Lookup("threadcreate").Count()
}

// cpu mem goroutine thread err.
func collect(cpuCore float64, memoryLimit uint64) (int, int, int, int, error) {
	cpu, mem, gNum, tNum, err := getUsage()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// The default percent is from all cores, multiply by cpu core
	// but it's inconvenient to calculate the proper percent
	// here we divide by core number, so we can set a percent bar more intuitively
	cpuPercent := cpu / cpuCore

	memPercent := float64(mem) / float64(memoryLimit) * 100

	return int(cpuPercent), int(memPercent), gNum, tNum, nil
}

func matchRule(history ring, curVal, ruleMin, ruleAbs, ruleDiff, ruleMax int) (bool, string) {
	// should bigger than rule min
	if curVal < ruleMin {
		return false, fmt.Sprintf("curVal [%d]< ruleMin [%d]", curVal, ruleMin)
	}

	// if ruleMax is enable and current value bigger max, skip dumping
	if ruleMax != NotSupportTypeMaxConfig && curVal >= ruleMax {
		return false, ""
	}

	// the current peak load exceed the absolute value
	if curVal > ruleAbs {
		return true, fmt.Sprintf("curVal [%d] > ruleAbs [%d]", curVal, ruleAbs)
	}

	// the peak load matches the rule
	avg := history.avg()
	if curVal >= avg*(100+ruleDiff)/100 {
		return true, fmt.Sprintf("curVal[%d] >= avg[%d]*(100+ruleDiff)/100", curVal, avg)
	}
	return false, ""
}

func getBinaryFileName(filePath string, dumpType configureType, eventID string) string {
	binarySuffix := time.Now().Format("20060102150405.000") + ".bin"

	return path.Join(filePath, type2name[dumpType]+"."+eventID+"."+binarySuffix)
}

func writeFile(data bytes.Buffer, dumpType configureType, dumpOpts *DumpOptions, eventID string) error {
	if dumpOpts.DumpProfileType == textDump {
		// write to log
		if dumpOpts.DumpFullStack {
			res := trimResult(data)
			return fmt.Errorf(res) // nolint:goerr113
		}
		return fmt.Errorf(data.String())
	}

	binFileName := getBinaryFileName(dumpOpts.DumpPath, dumpType, eventID)

	bf, err := os.OpenFile(binFileName, defaultLoggerFlags, defaultLoggerPerm) // nolint:gosec
	if err != nil {
		return fmt.Errorf("[Holmes] pprof %v write to file failed : %w", type2name[dumpType], err)
	}
	defer bf.Close() //nolint:errcheck,gosec

	if _, err = bf.Write(data.Bytes()); err != nil {
		return fmt.Errorf("[Holmes] pprof %v write to file failed : %w", type2name[dumpType], err)
	}
	return nil
}
