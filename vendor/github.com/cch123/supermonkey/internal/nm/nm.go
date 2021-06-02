// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nm

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"sort"

	"github.com/cch123/supermonkey/internal/objfile"
)

var (
	sortOrder = "size"
	printSize = false
	printType = false

	filePrefix = false
)

func init() {
	flag.Var(nflag(0), "n", "") // alias for -sort address
}

type nflag int

func (nflag) IsBoolFlag() bool {
	return true
}

func (nflag) Set(value string) error {
	if value == "true" {
		sortOrder = "address"
	}
	return nil
}

func (nflag) String() string {
	if sortOrder == "address" {
		return "true"
	}
	return "false"
}

// Parse parse the file and get the sym table
func Parse(filePath string) (string, error) {
	return nm(filePath)
}

var exitCode = 0

func errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
	exitCode = 1
}

func nm(file string) (string, error) {
	f, err := objfile.Open(file)
	if err != nil {
		errorf("%v", err)
		return "", err
	}
	defer f.Close()

	var result = bytes.Buffer{}
	w := bufio.NewWriter(&result)

	entries := f.Entries()

	var found bool

	for _, e := range entries {
		syms, err := e.Symbols()
		if err != nil {
			errorf("reading %s: %v", file, err)
		}
		if len(syms) == 0 {
			continue
		}

		found = true

		switch sortOrder {
		case "address":
			sort.Slice(syms, func(i, j int) bool { return syms[i].Addr < syms[j].Addr })
		case "name":
			sort.Slice(syms, func(i, j int) bool { return syms[i].Name < syms[j].Name })
		case "size":
			sort.Slice(syms, func(i, j int) bool { return syms[i].Size > syms[j].Size })
		}

		for _, sym := range syms {
			if len(entries) > 1 {
				name := e.Name()
				if name == "" {
					fmt.Fprintf(w, "%s(%s):\t", file, "_go_.o")
				} else {
					fmt.Fprintf(w, "%s(%s):\t", file, name)
				}
			} else if filePrefix {
				fmt.Fprintf(w, "%s:\t", file)
			}
			if sym.Code == 'U' {
				fmt.Fprintf(w, "%8s", "")
			} else {
				fmt.Fprintf(w, "%8x", sym.Addr)
			}
			if printSize {
				fmt.Fprintf(w, " %10d", sym.Size)
			}
			fmt.Fprintf(w, " %c %s", sym.Code, sym.Name)
			if printType && sym.Type != "" {
				fmt.Fprintf(w, " %s", sym.Type)
			}
			fmt.Fprintf(w, "\n")
		}
	}

	if !found {
		errorf("reading %s: no symbols", file)
	}

	w.Flush()
	return result.String(), nil
}
