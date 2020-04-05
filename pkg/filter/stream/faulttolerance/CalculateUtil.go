package faulttolerance

import (
	"fmt"
	"strconv"
)

func DivideUint64(numerator uint64, denominator uint64) float64 {
	a := float64(numerator)
	b := float64(denominator)
	value := a / b
	result, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return result
}

func DivideFloat64(numerator float64, denominator float64) float64 {
	value := numerator / denominator
	result, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return result
}
