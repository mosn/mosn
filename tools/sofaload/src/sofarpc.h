/*
 * nghttp2 - HTTP/2 C Library
 *
 * Copyright (c) 2015 British Broadcasting Corporation
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
 * LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
 * OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
 * WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */
#ifndef SOFARPC_H
#define SOFARPC_H

namespace h2load {

// ~~ constans
const char PROTOCOL_CODE_V1 = 1;
const char PROTOCOL_CODE_V2 = 2;

const char PROTOCOL_VERSION_1 = 1; // version
const char PROTOCOL_VERSION_2 = 2;

const int REQUEST_HEADER_LEN_V1 = 22; // protocol header fields length
const int REQUEST_HEADER_LEN_V2 = 24;

const int RESPONSE_HEADER_LEN_V1 = 20;
const int RESPONSE_HEADER_LEN_V2 = 22;

const int LESS_LEN_V1 = RESPONSE_HEADER_LEN_V1; // minimal length for decoding
const int LESS_LEN_V2 = RESPONSE_HEADER_LEN_V2;

const char RESPONSE = 0; // cmd type
const char REQUEST = 1;
const char REQUEST_ONEWAY = 2;

const uint16_t HEARTBEAT = 0; // cmd code
const uint16_t RPC_REQUEST = 1;
const uint16_t RPC_RESPONSE = 2;

const char HESSIAN2_SERIALIZE = 1; // serialize

const uint16_t RESPONSE_STATUS_SUCCESS = 0;                // 0x00 response status
const uint16_t RESPONSE_STATUS_ERROR = 1;                  // 0x01
const uint16_t RESPONSE_STATUS_SERVER_EXCEPTION = 2;       // 0x02
const uint16_t RESPONSE_STATUS_UNKNOWN = 3;                // 0x03
const uint16_t RESPONSE_STATUS_SERVER_THREADPOOL_BUSY = 4; // 0x04
const uint16_t RESPONSE_STATUS_ERROR_COMM = 5;             // 0x05
const uint16_t RESPONSE_STATUS_NO_PROCESSOR = 6;           // 0x06
const uint16_t RESPONSE_STATUS_TIMEOUT = 7;                // 0x07
const uint16_t RESPONSE_STATUS_CLIENT_SEND_ERROR = 8;      // 0x08
const uint16_t RESPONSE_STATUS_CODEC_EXCEPTION = 9;        // 0x09
const uint16_t RESPONSE_STATUS_CONNECTION_CLOSED = 16;     // 0x10
const uint16_t RESPONSE_STATUS_SERVER_SERIAL_EXCEPTION = 17;   // 0x11
const uint16_t RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION = 18; // 0x12

} // namespace h2load

#endif // SOFARPC_H