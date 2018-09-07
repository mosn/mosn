// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// gxzlib do zip compress/uncompress
package gxzlib

import (
	"bytes"
	"compress/zlib"
	"io"
)

import (
	"github.com/juju/errors"
)

// DoZlibCompresss do zip compress
func DoZlibCompress(src []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

// DoZlibUncompresss do unzip compress
func DoZlibUncompress(compressSrc []byte) ([]byte, error) {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, errors.Annotate(err, "fialed to unzip")
	}
	io.Copy(&out, r)
	return out.Bytes(), nil
}
