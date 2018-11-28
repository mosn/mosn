// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hessian

const (
	mask = byte(127)
	flag = byte(128)
)

const (
	TAG_READ        = int32(-1)
	ASCII_GAP       = 32
	CHUNK_SIZE      = 4096
	BC_BINARY       = byte('B') // final chunk
	BC_BINARY_CHUNK = byte('A') // non-final chunk

	BC_BINARY_DIRECT  = byte(0x20) // 1-byte length binary
	BINARY_DIRECT_MAX = byte(0x0f)
	BC_BINARY_SHORT   = byte(0x34) // 2-byte length binary
	BINARY_SHORT_MAX  = 0x3ff      // 0-1023 binary

	BC_DATE        = byte(0x4a) // 64-bit millisecond UTC date
	BC_DATE_MINUTE = byte(0x4b) // 32-bit minute UTC date

	BC_DOUBLE = byte('D') // IEEE 64-bit double

	BC_DOUBLE_ZERO  = byte(0x5b)
	BC_DOUBLE_ONE   = byte(0x5c)
	BC_DOUBLE_BYTE  = byte(0x5d)
	BC_DOUBLE_SHORT = byte(0x5e)
	BC_DOUBLE_MILL  = byte(0x5f)

	BC_FALSE = byte('F') // boolean false

	BC_INT = byte('I') // 32-bit int

	INT_DIRECT_MIN = -0x10
	INT_DIRECT_MAX = byte(0x2f)
	BC_INT_ZERO    = byte(0x90)

	INT_BYTE_MIN     = -0x800
	INT_BYTE_MAX     = 0x7ff
	BC_INT_BYTE_ZERO = byte(0xc8)

	BC_END = byte('Z')

	INT_SHORT_MIN     = -0x40000
	INT_SHORT_MAX     = 0x3ffff
	BC_INT_SHORT_ZERO = byte(0xd4)

	BC_LIST_VARIABLE         = byte(0x55)
	BC_LIST_FIXED            = byte('V')
	BC_LIST_VARIABLE_UNTYPED = byte(0x57)
	BC_LIST_FIXED_UNTYPED    = byte(0x58)

	BC_LIST_DIRECT         = byte(0x70)
	BC_LIST_DIRECT_UNTYPED = byte(0x78)
	LIST_DIRECT_MAX        = byte(0x7)

	BC_LONG         = byte('L') // 64-bit signed integer
	LONG_DIRECT_MIN = -0x08
	LONG_DIRECT_MAX = byte(0x0f)
	BC_LONG_ZERO    = byte(0xe0)

	LONG_BYTE_MIN     = -0x800
	LONG_BYTE_MAX     = 0x7ff
	BC_LONG_BYTE_ZERO = byte(0xf8)

	LONG_SHORT_MIN     = -0x40000
	LONG_SHORT_MAX     = 0x3ffff
	BC_LONG_SHORT_ZERO = byte(0x3c)

	BC_LONG_INT = byte(0x59)

	BC_MAP         = byte('M')
	BC_MAP_UNTYPED = byte('H')

	BC_NULL = byte('N') // x4e

	BC_OBJECT     = byte('O')
	BC_OBJECT_DEF = byte('C')

	BC_OBJECT_DIRECT  = byte(0x60)
	OBJECT_DIRECT_MAX = byte(0x0f)

	BC_REF = byte(0x51)

	BC_STRING       = byte('S') // final string
	BC_STRING_CHUNK = byte('R') // non-final string

	BC_STRING_DIRECT  = byte(0x00)
	STRING_DIRECT_MAX = byte(0x1f)
	BC_STRING_SHORT   = byte(0x30)
	STRING_SHORT_MAX  = 0x3ff

	BC_TRUE = byte('T')

	P_PACKET_CHUNK = byte(0x4f)
	P_PACKET       = byte('P')

	P_PACKET_DIRECT   = byte(0x80)
	PACKET_DIRECT_MAX = byte(0x7f)

	P_PACKET_SHORT   = byte(0x70)
	PACKET_SHORT_MAX = 0xfff
	ARRAY_STRING     = "[string"
	ARRAY_INT        = "[int"
	ARRAY_DOUBLE     = "[double"
	ARRAY_FLOAT      = "[float"
	ARRAY_BOOL       = "[boolean"
	ARRAY_LONG       = "[long"

	PATH_KEY      = "path"
	INTERFACE_KEY = "interface"
	VERSION_KEY   = "version"
	TIMEOUT_KEY   = "timeout"

	STRING_NIL   = "null"
	STRING_TRUE  = "true"
	STRING_FALSE = "false"
	STRING_ZERO  = "0.0"
	STRING_ONE   = "1.0"
)

//x00 - x1f    # utf-8 string length 0-32
//x20 - x2f    # binary data length 0-16
//x30 - x33    # utf-8 string length 0-1023
//x34 - x37    # binary data length 0-1023
//x38 - x3f    # three-octet compact long (-x40000 to x3ffff)
//x40          # reserved (expansion/escape)
//x41          # 8-bit binary data non-final chunk ('A')
//x42          # 8-bit binary data final chunk ('B')
//x43          # object type definition ('C')
//x44          # 64-bit IEEE encoded double ('D')
//x45          # reserved
//x46          # boolean false ('F')
//x47          # reserved
//x48          # untyped map ('H')
//x49          # 32-bit signed integer ('I')
//x4a          # 64-bit UTC millisecond date
//x4b          # 32-bit UTC minute date
//x4c          # 64-bit signed long integer ('L')
//x4d          # map with type ('M')
//x4e          # null ('N')
//x4f          # object instance ('O')
//x50          # reserved
//x51          # reference to map/list/object - integer ('Q')
//x52          # utf-8 string non-final chunk ('R')
//x53          # utf-8 string final chunk ('S')
//x54          # boolean true ('T')
//x55          # variable-length list/vector ('U')
//x56          # fixed-length list/vector ('V')
//x57          # variable-length untyped list/vector ('W')
//x58          # fixed-length untyped list/vector ('X')
//x59          # long encoded as 32-bit int ('Y')
//x5a          # list/map terminator ('Z')
//x5b          # double 0.0
//x5c          # double 1.0
//x5d          # double represented as byte (-128.0 to 127.0)
//x5e          # double represented as short (-32768.0 to 327676.0)
//x5f          # double represented as float
//x60 - x6f    # object with direct type
//x70 - x77    # fixed list with direct length
//x78 - x7f    # fixed untyped list with direct length
//x80 - xbf    # one-octet compact int (-x10 to x3f, x90 is 0)
//xc0 - xcf    # two-octet compact int (-x800 to x7ff)
//xd0 - xd7    # three-octet compact int (-x40000 to x3ffff)
//xd8 - xef    # one-octet compact long (-x8 to xf, xe0 is 0)
//xf0 - xff    # two-octet compact long (-x800 to x7ff, xf8 is 0)
