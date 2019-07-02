/*
 * sofaload
 *
 * Copyright (c) 2019 Ye Yongjie
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
#include "h2load_sofarpc_session.h"

#include <iostream>

namespace h2load {

SofaRpcSession::SofaRpcSession(Client *client)
    : client_(client), stream_req_counter_(1),
      read_buffer_(&client->worker->mcpool), bytes_to_discard_(0),
      last_stream_id_(-1), last_respstatus_(-1), terminate_(false) {}

SofaRpcSession::~SofaRpcSession() {}

void SofaRpcSession::on_connect() {
    // if (client_->worker->config->verbose) {
    //     std::cout << "on_connect" << std::endl;
    // }
    client_->signal_write();
}

int SofaRpcSession::submit_request() {
    auto config = client_->worker->config;
    const auto &req = config->sofarpcreqs[client_->reqidx];

    client_->reqidx++;
    if (client_->reqidx == config->sofarpcreqs.size()) {
        client_->reqidx = 0;
    }

    // if (client_->worker->config->verbose) {
    //     std::cout << "submit_request" << std::endl;
    // }

    int stream_id = stream_req_counter_++;
    client_->on_request(stream_id);

    auto req_stat = client_->get_req_stat(stream_id);
    client_->record_request_time(req_stat);

    // a piece of shit
    char *bytes = new char[req.size()];
    std::memcpy(bytes, req.c_str(), req.size());
    util::putBigEndianI32(&bytes[5], stream_id);

    client_->wb.append(bytes, req.size());
    delete[] bytes;

    // std::cout << "[submit_request] " << stream_id << " " << req << std::endl;

    return 0;
}

int SofaRpcSession::on_read(const uint8_t *data, size_t len) {

    if (client_->worker->config->verbose) {
        // std::cout << "--on_read--" << std::endl;
        std::cout.write(reinterpret_cast<const char *>(data), len);
        // std::cout << "--on_read--" << std::endl;
    }

    read_buffer_.append(data, len);

    for (;;) {
        if (bytes_to_discard_ != 0) {
            client_->record_ttfb();
            // if (read_buffer_.rleft() < bytes_to_discard_)
            //     break;
            // read_buffer_.drain(bytes_to_discard_);
            size_t bytes_to_drain = std::min(bytes_to_discard_, read_buffer_.rleft());
            read_buffer_.drain(bytes_to_drain);
            client_->worker->stats.bytes_body += bytes_to_drain;
            bytes_to_discard_ -= bytes_to_drain;

            // TODO
            auto req_stat = client_->get_req_stat(last_stream_id_);
            if (req_stat->data_offset >= client_->worker->config->data_length) {
                client_->on_stream_close(last_stream_id_, last_respstatus_ == RESPONSE_STATUS_SUCCESS);
            }
        }
        if (read_buffer_.rleft() >= RESPONSE_HEADER_LEN_V1) {
            char bytes[RESPONSE_HEADER_LEN_V1];
            read_buffer_.remove(bytes, RESPONSE_HEADER_LEN_V1);

            int32_t requestId = util::getBigEndianI32(&bytes[5]);
            uint16_t respstatus = util::getBigEndianI16(&bytes[10]);
            uint16_t classLen = util::getBigEndianI16(&bytes[12]);
            uint16_t headerLen = util::getBigEndianI16(&bytes[14]);
            uint32_t contentLen = util::getBigEndianI32(&bytes[16]);
            // std::cout << requestId << " " << respstatus << " " << classLen << " " << headerLen << " " << contentLen << std::endl;

            last_stream_id_ = requestId;
            last_respstatus_ = respstatus;

            client_->on_sofarpc_status(requestId, respstatus);

            client_->worker->stats.bytes_head += RESPONSE_HEADER_LEN_V1;
            client_->worker->stats.bytes_head_decomp += RESPONSE_HEADER_LEN_V1;

            int bodyLen = classLen + headerLen + contentLen;
            bytes_to_discard_ = bodyLen;
        } else {
            break;
        }
    }

    return 0;
}

int SofaRpcSession::on_write() {

    // if (client_->worker->config->verbose) {
    //     std::cout << "on_write" << std::endl;
    // }

    if (terminate_ && client_->wb.rleft() == 0) {
        return -1;
    }

    return 0;
}

void SofaRpcSession::terminate() {
    // if (client_->worker->config->verbose) {
    //     std::cout << "terminate" << std::endl;
    // }
    terminate_ = true;
}

size_t SofaRpcSession::max_concurrent_streams() {
    return (size_t)client_->worker->config->max_concurrent_streams;
}

} // namespace h2load
