// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxrand encapsulates some golang.math.rand functions.
package gxrand

import (
	"fmt"
	"math/rand"
	"time"
)

import (
	"github.com/juju/errors"
)

// CalRedPacket provides tencent red packet algorithm
// MOD: 2016-06-01 23:03
func CalRedPacket(total int32, num int32) ([]int32, error) {
	if num < 1 || total < 1 {
		return nil, errors.New(fmt.Sprintf("@num{%v}, @total{%v}", num, total))
	}

	var (
		i       int32   = 0
		k       int32   = 0
		min     int32   = 1
		max     int32   = 0
		balance int32   = total
		money   int32   = 0
		packet  []int32 = make([]int32, num)
	)

	rand.Seed(time.Now().Unix())
	for i = 1; i < num; i++ {
		max = balance - min*(num-i)
		k = int32((num - i) / 2)
		if num-i <= 2 {
			k = num - i
		}
		max = max / k
		if max <= min {
			max = min + 1
		}
		// fmt.Println(min, ",", max, ",", max-min)
		money = int32(rand.Intn(int(max-min))) + min
		balance = balance - money
		// fmt.Printf("idx:%d, money:%d, balance:%d\n", i, money, balance)
		packet[i-1] = money
	}
	packet[i-1] = balance
	// money = balance
	// balance = 0
	// fmt.Printf("idx:%d, packet money:%d, sumary balance:%d\n", i, money, balance)

	return packet, nil
}
