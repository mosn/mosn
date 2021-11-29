package supermonkey

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/cch123/supermonkey/internal/bouk"
	"github.com/cch123/supermonkey/internal/nm"
)

type PatchGuard = bouk.PatchGuard

var (
	symbolTable = map[string]uintptr{}
)

// Patch replaces a function with another
func Patch(target, replacement interface{}) *PatchGuard {
	return bouk.Patch(target, replacement)
}

func PatchInstanceMethod(target reflect.Type, methodName string, replacement interface{}) *PatchGuard {
	return bouk.PatchInstanceMethod(target, methodName, replacement)
}

// PatchByFullSymbolName needs user to provide the full symbol path
func PatchByFullSymbolName(symbolName string, patchFunc interface{}) *PatchGuard {
	addr := symbolTable[symbolName]
	if addr == 0 {
		prompt := fmt.Sprintf(`The symbol is %v, and the patch target addr is 0, there may be 2 possible reasons:
  1. the function is inlined, please add //go:noinline to function comment or add -l to gcflags
  2. your input for symbolName or pkg/obj/method is wrong, check by using go tool nm {your_bin_file} %s`, symbolName, "\n")
		colorPrint(prompt, green)
		similarSymbols(symbolName)
		os.Exit(1)
	}
	return bouk.PatchSymbol(unsafe.Pointer(addr), patchFunc)
}

// UnpatchAll unpatches all functions
func UnpatchAll() {
	bouk.UnpatchAll()
}

func init() {
	content, _ := nm.Parse(os.Args[0])

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line := strings.TrimSpace(line)
		arr := strings.Split(line, " ")
		if len(arr) < 3 {
			continue
		}

		funcSymbol, addr := arr[2], arr[0]
		addrUint, _ := strconv.ParseUint(addr, 16, 64)
		symbolTable[funcSymbol] = uintptr(addrUint)
	}
}

// Finds similar symbols
func similarSymbols(symbolName string) {
	similarList := make([]string, 0)
	for s, _ := range symbolTable {
		if strings.Contains(s, symbolName) {
			similarList = append(similarList, "  - "+s)
		}
	}

	if len(similarList) == 0 {
		return
	}

	if len(similarList) == 1 {
		colorPrint("The most similar symbol is", red)
	} else if len(similarList) > 1 {
		colorPrint("The most similar symbols are", red)
	}
	colorPrint(strings.Join(similarList, "\n"), red)
}

const (
	red   = 31
	green = 32
)

func colorPrint(s string, f int) {
	fmt.Printf("\033[0;%dm%s\033[0m\n", f, s)
}
