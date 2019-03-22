package tools

import (
	"strconv"
	"strings"
	"unicode"
)

const (
	// B byte
	B uint64 = 1
	// K kilobyte
	K uint64 = 1 << (10 * iota)
	// M megabyte
	M
	// G gigabyte
	G
	// T TeraByte
	T
	// P PetaByte
	P
	// E ExaByte
	E
)

var unitMap = map[string]uint64{
	"B":  B,
	"K":  K,
	"KB": K,
	"M":  M,
	"MB": M,
	"G":  G,
	"GB": G,
	"T":  T,
	"TB": T,
	"P":  P,
	"PB": P,
	"E":  E,
	"EB": E,
}

// ParseLogSizeMb : Translate x(B),xKB,xMB... to uint64 x(MB)
func ParseLogSizeMb(oriSize string) (ret uint64) {
	var defaultLogSizeMb uint64 = 100
	if oriSize == "" {
		return defaultLogSizeMb
	}
	iLogSize, err := strconv.ParseUint(oriSize, 10, 64)
	if err == nil {
		ret = iLogSize / 1024 / 1024
		if ret == 0 {
			return defaultLogSizeMb
		}
		return ret
	}
	sLogSize := ""
	sUnit := ""
	for idx, c := range oriSize {
		if !unicode.IsDigit(c) {
			sLogSize = strings.TrimSpace(oriSize[:idx])
			sUnit = strings.TrimSpace(oriSize[idx:])
			break
		}
	}
	if sLogSize == "" {
		return defaultLogSizeMb
	}
	iLogSize, err = strconv.ParseUint(sLogSize, 10, 64)
	if err != nil {
		return defaultLogSizeMb
	}
	sUnit = strings.ToUpper(sUnit)
	iUnit, exists := unitMap[sUnit]
	if !exists {
		return defaultLogSizeMb
	}
	ret = iLogSize * iUnit / 1024 / 1024
	if ret == 0 {
		ret = defaultLogSizeMb
	}
	return ret
}

// ParseLogNum : Parse uint64 from string
func ParseLogNum(strVal string) (ret uint64) {
	var defualtLogNum uint64 = 10
	ret, err := strconv.ParseUint(strVal, 10, 64)
	if err != nil {
		return defualtLogNum
	}
	return ret
}
