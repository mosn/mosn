// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxioutil implements some I/O utility functions.
package gxioutil

import (
	"bufio"
	"io/ioutil"
	"os"
)

import (
	"github.com/juju/errors"
)

func ReadFile(file string) ([]byte, error) {
	if len(file) == 0 {
		return nil, errors.Errorf("@file is nil")
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, errors.Errorf("@file{%s} not found", file)
	}
	f, err := os.OpenFile(file, os.O_RDONLY, 0666)
	if err != nil {
		return nil, errors.Annotatef(err, "file:%s", file)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	data, err := ioutil.ReadAll(reader)
	if len(data) == 0 {
		return nil, errors.Errorf("can not get content of @file{%s}", file)
	}

	return data, nil
}

func WriteFile(file string, data []byte) error {
	return ioutil.WriteFile(file, data, 0644)
}
