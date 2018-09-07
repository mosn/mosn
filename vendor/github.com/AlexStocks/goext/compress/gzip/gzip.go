// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// gxgzip do gzip compress/uncompress
package gxgzip

import (
	"bytes"
	"compress/gzip"
	"io"
)

import (
	"github.com/juju/errors"
)

// DoGzipCompresss do gzip compress
func DoGzipCompress(src []byte) []byte {
	var in bytes.Buffer
	w := gzip.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

// DoGzipUncompresss do gzip uncompress
func DoGzipUncompress(compressSrc []byte) ([]byte, error) {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, errors.Annotate(err, "fialed to ungzip")
	}
	io.Copy(&out, r)
	return out.Bytes(), nil
}
