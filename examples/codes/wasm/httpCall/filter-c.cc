// Copyright 2016-2020 Envoy Project Authors
// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include <string>
#include <string_view>
#include <unordered_map>

#include "proxy_wasm_intrinsics.h"

class ExampleContext : public Context {
public:
  explicit ExampleContext(uint32_t id, RootContext *root) : Context(id, root) {}

  FilterHeadersStatus onRequestHeaders(uint32_t headers, bool end_of_stream) override;
};

static RegisterContextFactory register_ExampleContext(CONTEXT_FACTORY(ExampleContext));

void ExampleHttpCallback(uint32_t num_headers, size_t body_size, uint32_t num_trailers) {
  auto result = getHeaderMapPairs(WasmHeaderMapType::HttpCallResponseHeaders);
  auto pairs = result->pairs();
  LOG_INFO(std::string("headers: ") + std::to_string(pairs.size()));
  for (auto &p : pairs) {
    LOG_INFO(std::string(p.first) + std::string(" -> ") + std::string(p.second));
  }

  auto body = getBufferBytes(WasmBufferType::HttpCallResponseBody, 0, body_size);
  LOG_ERROR(std::string("body: ") + std::string(body->view()));
}

FilterHeadersStatus ExampleContext::onRequestHeaders(uint32_t, bool) {
  HeaderStringPairs headers;
  HeaderStringPairs trailers;

  LOG_INFO("before http call");
  this->root()->httpCall("http://127.0.0.1:2046/", headers, "", trailers, 50000, ExampleHttpCallback);
  LOG_INFO("after http call");

  return FilterHeadersStatus::Continue;
}