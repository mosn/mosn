package utils

import (
	"reflect"
	"sort"
	"strings"
)

func StringIn(s string, ss []string) bool {
	for _, v := range ss {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

func SliceEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	return reflect.DeepEqual(a, b)
}
