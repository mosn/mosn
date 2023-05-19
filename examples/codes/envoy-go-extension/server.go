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

package main

import (
	"fmt"
	"net/http"
)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[UPSTREAM]receive request %s\n", r.URL)
	for k, v := range r.Header {
		fmt.Printf("%v -> %v\n", k, v)
	}

	w.Header().Add("from", "external http server")
	w.Write([]byte("response body from external http server"))
}

func main() {
	http.HandleFunc("/", ServeHTTP)
	http.ListenAndServe("127.0.0.1:8090", nil)
}
