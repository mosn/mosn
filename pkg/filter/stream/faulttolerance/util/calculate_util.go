package util

import (
	"fmt"
	"strconv"
)

func DivideInt64(numerator int64, denominator int64) float64 {
	a := float64(numerator)
	b := float64(denominator)
	return DivideFloat64(a, b)
}

func DivideFloat64(numerator float64, denominator float64) float64 {
	value := numerator / denominator
	result, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return result
}
