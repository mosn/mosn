// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxrand encapsulates some golang.math.rand functions
package gxrand

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// refer to https://github.com/gorilla/feeds/blob/master/uuid.go
type UUID [16]byte

// Constructs a new V4 (random) UUID.
func NewUUID() *UUID {
	u := &UUID{}
	_, err := rand.Read(u[:16])
	if err != nil {
		panic(err)
	}

	u[8] = (u[8] | 0x80) & 0xBf
	u[6] = (u[6] | 0x40) & 0x4f
	return u
}

func (u *UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func (u *UUID) HexString() string {
	return hex.EncodeToString(u[:16])
}
