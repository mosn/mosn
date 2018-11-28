# Circular Buffer

[![GoDoc](https://godoc.org/github.com/ashishgandhi/buffer?status.svg)](https://godoc.org/github.com/ashishgandhi/buffer)
[![Build Status](https://travis-ci.org/ashishgandhi/buffer.svg?branch=master)](https://travis-ci.org/ashishgandhi/buffer)
[![Coverage Status](https://codecov.io/github/ashishgandhi/buffer/coverage.svg?branch=master)](https://codecov.io/github/ashishgandhi/buffer?branch=master)

This is a circular buffer implementation designed to be used with gigabyte sized buffers. It uses a file as its backing store. The backing file is `mmap`'d into memory. One can lower the resident memory size of the program by advising where the customer cursor is (uses `madvise`). The buffer can be loaded again from the backing file if the program exists. 

For now please check tests for examples.
