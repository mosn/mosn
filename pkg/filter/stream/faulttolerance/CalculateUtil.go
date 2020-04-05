package faulttolerance

import (
	"fmt"
	"strconv"
)

func Divide(numerator uint64, denominator uint64) float64 {
	a := float64(numerator)
	b := float64(denominator)
	value := a / b
	result, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return result
}
