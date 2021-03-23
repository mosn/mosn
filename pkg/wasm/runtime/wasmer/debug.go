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

package wasmer

import (
	"bytes"
	"debug/dwarf"
	"errors"
	"fmt"

	"mosn.io/mosn/pkg/log"
)

const (
	SecCustom   byte = 0
	SecType     byte = 1
	SecImport   byte = 2
	SecFunction byte = 3
	SecTable    byte = 4
	SecMemory   byte = 5
	SecGlobal   byte = 6
	SecExport   byte = 7
	SecStart    byte = 8
	SecElement  byte = 9
	SecCode     byte = 10
	SecData     byte = 11
)

var secTypeNames = map[byte]string{
	SecCustom:   "CUSTOM",
	SecType:     "TYPE",
	SecImport:   "IMPORT",
	SecFunction: "FUNCTION",
	SecTable:    "TABLE",
	SecMemory:   "MEMORY",
	SecGlobal:   "GLOBAL",
	SecExport:   "EXPORT",
	SecStart:    "START",
	SecElement:  "ELEMENT",
	SecCode:     "CODE",
	SecData:     "DATA",
}

type dwarfInfo struct {
	data              *dwarf.Data
	codeSectionOffset int
	lineReader        *dwarf.LineReader
}

func (d *dwarfInfo) getLineReader() *dwarf.LineReader {
	if d.lineReader != nil {
		d.lineReader.Reset()
		return d.lineReader
	}

	// Get the line table for the first CU.
	cu, err := d.data.Reader().Next()
	if err != nil {
		log.DefaultLogger.Errorf("[wasmer][debug] getLineReader fail to do d.Reader().Next(), err: %v", err)
	}

	lineReader, err := d.data.LineReader(cu)
	if err != nil {
		log.DefaultLogger.Errorf("[wasmer][debug] getLineReader fail to do d.LineReader(), err: %v", err)
	}

	d.lineReader = lineReader

	return d.lineReader
}

func (d *dwarfInfo) SeekPC(pc uint64) *dwarf.LineEntry {
	var line dwarf.LineEntry

	lr := d.getLineReader()
	if lr == nil {
		return nil
	}

	// https://yurydelendik.github.io/webassembly-dwarf/#pc
	pc -= uint64(d.codeSectionOffset)

	if err := lr.SeekPC(pc, &line); err != nil {
		return nil
	}

	return &line
}

type CustomSection struct {
	name string
	data []byte
}

func parseDwarf(data []byte) *dwarfInfo {
	customSections, codeSectionOffset, err := loadCustomSections(data)
	if err != nil {
		return nil
	}

	dwarfData, err := dwarf.New(
		customSections[".debug_abbrev"].data,
		customSections[".debug_aranges"].data,
		customSections[".debug_frame"].data,
		customSections[".debug_info"].data,
		customSections[".debug_line"].data,
		customSections[".debug_pubnames"].data,
		customSections[".debug_ranges"].data,
		customSections[".debug_str"].data)
	if err != nil {
		return nil
	}

	return &dwarfInfo{
		data:              dwarfData,
		codeSectionOffset: codeSectionOffset,
	}
}

func loadCustomSections(data []byte) (map[string]CustomSection, int, error) {
	// Index of first section
	i := 8
	codeSectionOffset := 0

	// Skip preamble
	if !bytes.Equal(data[0:i], []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}) {
		return nil, 0, errors.New("invalid preamble. Only Wasm v1 files are supported")
	}

	customSections := map[string]CustomSection{}
	// Read all sections
	for i < len(data) {
		sectionType := data[i]
		if _, ok := secTypeNames[sectionType]; !ok {
			return nil, 0, fmt.Errorf("section type 0x%02X not supported", sectionType)
		}
		i++

		sectionDataSize, n := readULEB128(data[i:])
		i += n

		sectionName := ""

		if sectionType == SecCustom {
			customSection := NewCustomSection(data[i : i+sectionDataSize])
			sectionName = customSection.name
			customSections[customSection.name] = customSection
		}

		if sectionType == SecCode {
			codeSectionOffset = i
		}

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			if len(sectionName) > 0 {
				log.DefaultLogger.Debugf("[wasmer][debug] loadCustomSections %v(%X) %v: data start at 0x%X, data size 0x%X",
					secTypeNames[sectionType], sectionType, sectionName, i, sectionDataSize)
			} else {
				log.DefaultLogger.Debugf("[wasmer][debug] loadCustomSections %v(%X): data start at 0x%X, data size 0x%X",
					secTypeNames[sectionType], sectionType, i, sectionDataSize)
			}
		}

		i += sectionDataSize
	}

	return customSections, codeSectionOffset, nil
}

// NewCustomSection properly read name and data from a custom section.
func NewCustomSection(data []byte) CustomSection {
	nameLength, n := readULEB128(data)

	return CustomSection{
		name: string(data[n : n+nameLength]),
		data: data[nameLength+n:],
	}
}

// Read a uLEB128 at the starting position. Returns (number read, number of bytes read).
func readULEB128(data []byte) (int, int) {
	length := 1
	for data[length-1]&0x80 > 0 {
		length++
	}

	n := int(DecodeULeb128(data[:length+1]))

	return n, length
}

// DecodeULeb128 decodes an unsigned LEB128 value to an unsigned int32 value. Returns the result as a uint32.
func DecodeULeb128(value []byte) uint32 {
	var result uint32
	var ctr uint
	var cur byte = 0x80

	for (cur&0x80 == 0x80) && ctr < 5 {
		cur = value[ctr] & 0xff
		result += uint32(cur&0x7f) << (ctr * 7)
		ctr++
	}

	return result
}
