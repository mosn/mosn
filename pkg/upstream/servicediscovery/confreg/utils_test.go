package registry

import (
    "testing"
    "fmt"
)

func Test_utils(t *testing.T) {
    fmt.Println(CalRetreatTime(0, 5))
    fmt.Println(CalRetreatTime(1, 5))
    fmt.Println(CalRetreatTime(2, 5))
    fmt.Println(CalRetreatTime(3, 5))
    fmt.Println(CalRetreatTime(4, 5))
    fmt.Println(CalRetreatTime(5, 5))
    fmt.Println(CalRetreatTime(6, 5))
}
