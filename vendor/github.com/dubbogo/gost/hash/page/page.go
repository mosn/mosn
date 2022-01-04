/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gxpage

// Page is the default implementation of Page interface
type Page struct {
	requestOffset int
	pageSize      int
	totalSize     int
	data          []interface{}
	totalPages    int
	hasNext       bool
}

// GetOffSet will return the offset
func (d *Page) GetOffset() int {
	return d.requestOffset
}

// GetPageSize will return the page size
func (d *Page) GetPageSize() int {
	return d.pageSize
}

// GetTotalPages will return the number of total pages
func (d *Page) GetTotalPages() int {
	return d.totalPages
}

// GetData will return the data
func (d *Page) GetData() []interface{} {
	return d.data
}

// GetDataSize will return the size of data.
// it's len(GetData())
func (d *Page) GetDataSize() int {
	return len(d.GetData())
}

// HasNext will return whether has next page
func (d *Page) HasNext() bool {
	return d.hasNext
}

// HasData will return whether this page has data.
func (d *Page) HasData() bool {
	return d.GetDataSize() > 0
}

// NewPage will create an instance
func NewPage(requestOffset int, pageSize int,
	data []interface{}, totalSize int) *Page {

	remain := totalSize % pageSize
	totalPages := totalSize / pageSize
	if remain > 0 {
		totalPages++
	}

	hasNext := totalSize-requestOffset-pageSize > 0

	return &Page{
		requestOffset: requestOffset,
		pageSize:      pageSize,
		data:          data,
		totalSize:     totalSize,
		totalPages:    totalPages,
		hasNext:       hasNext,
	}
}
