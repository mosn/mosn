;; wasi prints HTTP requests and responses to the console using `fd_write`.
(module $wasi
  ;; fd_write write bytes to a file descriptor.
  ;;
  ;; See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_write
  (import "wasi_snapshot_preview1" "fd_write" (func $fd_write
    (param $fd i32) (param $iovs i32) (param $iovs_len i32) (param $result.size i32)
    (result (;errno;) i32)))

  ;; enable_features tries to enable the given features and returns the entire
  ;; feature bitflag supported by the host.
  (import "http_handler" "enable_features" (func $enable_features
    (param $enable_features i32)
    (result (; enabled_features ;) i32)))

  ;; get_method writes the method to memory if it isn't larger than $buf_limit.
  ;; The result is its length in bytes. Ex. "GET"
  (import "http_handler" "get_method" (func $get_method
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_uri writes the URI to memory if it isn't larger than $buf_limit.
  ;; The result is its length in bytes. Ex. "/v1.0/hi?name=panda"
  (import "http_handler" "get_uri" (func $get_uri
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_protocol_version writes the HTTP protocol version to memory if it
  ;; isn't larger than `buf_limit`. The result is its length in bytes.
  ;; Ex. "HTTP/1.1"
  (import "http_handler" "get_protocol_version" (func $get_protocol_version
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; get_header_names writes all header names for the given $kind,
  ;; NUL-terminated, to memory if the encoded length isn't larger than
  ;; $buf_limit. The result is regardless of whether memory was written.
  (import "http_handler" "get_header_names" (func $get_header_names
    (param $kind i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; count << 32| len ;) i64)))

  ;; get_header_values writes all header names of the given $kind and $name,
  ;; NUL-terminated, to memory if the encoded length isn't larger than
  ;; $buf_limit. The result is regardless of whether memory was written.
  (import "http_handler" "get_header_values" (func $get_header_values
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $buf i32) (param $buf_limit i32)
    (result (; count << 32| len ;) i64)))

  ;; read_body reads up to $buf_limit bytes remaining in the $kind body into
  ;; memory at offset $buf. A zero $buf_limit will panic.
  ;;
  ;; The result is `0 or EOF(1) << 32|len`, where `len` is the length in bytes
  ;; read.
  (import "http_handler" "read_body" (func $read_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)
    (result (; 0 or EOF(1) << 32 | len ;) i64)))

  ;; write_body reads $buf_len bytes at memory offset `buf` and writes them to
  ;; the pending $kind body.
  (import "http_handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  ;; get_status_code returnts the status code produced by $next.
  (import "http_handler" "get_status_code" (func $get_status_code
    (result (; status_code ;) i32)))

  ;; http_handler guests are required to export "memory", so that imported
  ;; functions like "fd_write" can read memory.
  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; error starts at 32 to allow scratch space for WASI.
  (global $error i32 (i32.const 32))
  (data (i32.const 32) "host error")
  (global $error_len i32 (i32.const 10))

  ;; heap_start starts after $error
  (global $heap_start i32 (i32.const 64))

  (global $buf_limit i32 (i32.const 2048))
  (func $buf_remaining (param $buf i32) (result i32)
    (i32.sub (global.get $buf_limit) (local.get $buf)))

  (func $print_request_headers
    (call $print_headers (i32.const 0))) ;; header_kind_request
  (func $print_request_body
    (call $print_body (i32.const 0)))    ;; body_kind_request
  (func $print_request_trailers
    (call $print_headers (i32.const 2))) ;; header_kind_request_trailers
  (func $print_response_headers
    (call $print_headers (i32.const 1))) ;; header_kind_response
  (func $print_response_body
    (call $print_body (i32.const 1)))    ;; body_kind_response
  (func $print_response_trailers
    (call $print_headers (i32.const 3))) ;; header_kind_response_trailers

  ;; We don't require the trailers features as it defaults to no-op when
  ;; unsupported.
  ;;
  ;; required_features := feature_buffer_request|feature_buffer_response
  (global $required_features i32 (i32.const 3))

  ;; eof is the upper 32-bits of the $read_body result on EOF.
  (global $eof i64 (i64.const 4294967296)) ;; `1<<32|0`

  ;; must_enable_buffering ensures we can inspect request and response bodies
  ;; without interfering with the next handler.
  (func $must_enable_buffering
    (local $enabled_features i32)

    ;; enabled_features := enable_features(required_features)
    (local.set $enabled_features
      (call $enable_features (global.get $required_features)))

    ;; if enabled_features&required_features == 0 { panic }
    (if (i32.eqz (i32.and
          (local.get $enabled_features)
          (global.get $required_features)))
      (then unreachable)))

  ;; newline is the position in memory of '\n'
  (global $newline i32 (i32.const 0))

  ;; ignored is the position for fd_write to write data.
  (global $ignored i32 (i32.const 4))

  ;; iovecs is the position that begins a 2-entry I/O vector, where the second
  ;; is the predefined newline.
  (global $iovs i32 (i32.const 16))

  (func $init_wasi
    ;; store newline at buf $newline
    (i32.store8 (global.get $newline) (i32.const (; '\n'== ;) 10))

    ;; create a predefined 8-byte I/O vector for the newline. This is the
    ;; second I/O vector starting at $iovs.
    (i32.store (i32.const 24) (global.get $newline))
    (i32.store (i32.const 28) (i32.const  1))) ;; len of '\n'

  (func $main
    (call $must_enable_buffering)
    (call $init_wasi))

  (start $main)

  ;; handle_request prints the request message to the consolt and returns
  ;; non-zero to call the next handler.
  (func (export "handle_request") (result (; ctx_next ;) i64)
    ;; Print the incoming request to the console.
    (call $print_request_line)
    (call $print_request_headers)
    (call $print_request_body)
    (call $print_request_trailers)

    ;; uint32(ctx_next) == 1 means proceed to the next handler on the host.
    (return (i64.const 1)))

  ;; handle_response prints the response message or the word "error" to the
  ;; console.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32)

    ;; println("")
    (call $println (global.get $heap_start) (i32.const 0))

    ;; if is_error == 1 { println("host error"); return }
    (if (i32.eq (local.get $is_error) (i32.const 1))
      (then (call $println (global.get $error) (global.get $error_len))
            (return)))

    ;; Because we enabled buffering, we can read the response.
    ;; Print it to the console.
    (call $print_response_line)
    (call $print_response_headers)
    (call $print_response_body)
    (call $print_response_trailers))

  ;; $print_request_line prints the request line to the console.
  ;; Ex "GET /a HTTP/1.1"
  (func $print_request_line
    ;; buf is the current position in memory, initially $heap_start.
    (local $buf i32)

    ;; len is a temporary variable used for function results.
    (local $len i32)

    (local.set $buf (global.get $heap_start))

    ;; mem[buf:len] = method
    (local.set $len
      (call $get_method (local.get $buf) (call $buf_remaining (local.get $buf))))

    ;; buf += len
    (local.set $buf (i32.add (local.get $buf) (local.get $len)))

    ;; mem[buf++] = ' '
    (local.set $buf (call $store_space (local.get $buf)))

    ;; mem[buf:len] = get_uri
    (local.set $len
      (call $get_uri (local.get $buf) (call $buf_remaining (local.get $buf))))

    ;; buf += len
    (local.set $buf (i32.add (local.get $buf) (local.get $len)))

    ;; mem[buf++] = ' '
    (local.set $buf (call $store_space (local.get $buf)))

    ;; mem[buf:len] = get_protocol_version
    (local.set $len
      (call $get_protocol_version (local.get $buf) (call $buf_remaining (local.get $buf))))

    ;; buf += len
    (local.set $buf (i32.add (local.get $buf) (local.get $len)))

    ;; println(mem[heap_start:buf - heap_start])
    (call $println
      (global.get $heap_start)
      (i32.sub (local.get $buf) (global.get $heap_start))))

  ;; print_response_line prints the response line to the console, without the
  ;; status reason. Ex. "HTTP/1.1 200"
  (func $print_response_line
    ;; buf is the current position in memory, initially $heap_start.
    (local $buf i32)

    ;; len is a temporary variable used for function results.
    (local $len i32)

    (local.set $buf (global.get $heap_start))

    ;; mem[buf:len]  get_protocol_version
    (local.set $len
      (call $get_protocol_version (local.get $buf) (call $buf_remaining (local.get $buf))))

    ;; buf += len
    (local.set $buf (i32.add (local.get $buf) (local.get $len)))

    ;; mem[buf++] = ' '
    (local.set $buf (call $store_space (local.get $buf)))

    (call $store_status_code (local.get $buf) (call $get_status_code))

    ;; buf += 3
    (local.set $buf (i32.add (local.get $buf) (i32.const 3)))

    ;; println(mem[heap_start:buf - heap_start])
    (call $println
      (global.get $heap_start)
      (i32.sub (local.get $buf) (global.get $heap_start))))

  ;; $print_headers prints each header field to the console. Ex "a: b"
  (func $print_headers (param $kind i32)
    ;; buf is the current position in memory, initially $heap_start.
    (local $buf i32)

    ;; result is count << 32| len
    (local $result i64)

    ;; names_len is the length of all NUL-terminated names.
    (local $names_len i32)

    ;; buf_console is where the print function can begin writing.
    (local $buf_console i32)

    ;; name is the position of the current name.
    (local $name i32)

    ;; len is the length of the current NUL-terminated name, exclusive of NUL.
    (local $len i32)

    (local.set $buf (global.get $heap_start))

    ;; result = get_header_names(kind, buf, buf_limit)
    (local.set $result
      (call $get_header_names
        (local.get  $kind)
        (local.get  $buf)
        (global.get $buf_limit)))

    ;; if there are no headers, return
    (if (i64.eqz (local.get $result)) (then (return)))

    ;; names_len = uint32(result)
    (local.set $names_len (i32.wrap_i64 (local.get $result)))

    ;; if names_len > buf_limit { panic }
    (if (i32.gt_u (local.get $names_len) (global.get $buf_limit))
       (then (unreachable))) ;; too big so wasn't written

    ;; We can start writing memory after the NUL-terminated header names.
    (local.set $buf_console (i32.add (local.get $buf) (local.get $names_len)))

    ;; loop while we can read a NUL-terminated name.
    (loop $names
      ;; if mem[buf] == NUL
      (if (i32.eqz (i32.load8_u (local.get $buf)))
        (then ;; reached the end of the name

          ;; name = buf -len
          (local.set $name (i32.sub (local.get $buf) (local.get $len)))

          ;; print_header_fields(kind, name, buf_console, buf_limit)
          (call $print_header_fields
            (local.get  $kind)
            (local.get  $name) (local.get $len)
            (local.get  $buf_console)
            (global.get $buf_limit))

          (local.set $buf (i32.add (local.get $buf) (i32.const 1))) ;; buf++
          (local.set $len (i32.const 0))) ;; len = 0
        (else
          (local.set $len (i32.add (local.get $len) (i32.const 1))) ;; len++
          (local.set $buf (i32.add (local.get $buf) (i32.const 1))))) ;; buf++

      (local.set $names_len (i32.sub (local.get $names_len) (i32.const 1))) ;; names_len--

      ;; if names_len > 0 { continue } else { break }
      (br_if $names (i32.gt_u (local.get $names_len) (i32.const 0)))))

  ;; print_header_fields prints each header field to the console.
  (func $print_header_fields
    (param $kind i32)
    (param $name i32) (param  $name_len i32)
    (param  $buf i32) (param $buf_limit i32)

    ;; result is count << 32| len
    (local $result i64)

    ;; values_len is the length of all NUL-terminated values.
    (local $values_len i32)

    ;; buf_console is where the print function can begin writing.
    (local $buf_console i32)

    ;; value is the position of the current name.
    (local $value i32)

    ;; len is the length of the current NUL-terminated value, exclusive of NUL.
    (local $len i32)

    ;; result = get_header_values(kind, mem[name:name_len], mem[buf:buf_limit])
    (local.set $result (call $get_header_values
      (local.get $kind)
      (local.get $name) (local.get $name_len)
      (local.get $buf)  (local.get $buf_limit)))

    ;; values_len = uint32(result)
    (local.set $values_len (i32.wrap_i64 (local.get $result)))

    ;; if values_len > buf_limit { panic }
    (if (i32.gt_u (local.get $values_len) (global.get $buf_limit))
       (then (unreachable))) ;; too big so wasn't written

    ;; buf_console = buf + values_len
    (local.set $buf_console (i32.add (local.get $buf) (local.get $values_len)))

    ;; loop while we can read a NUL-terminated value.
    (loop $values
      ;; if mem[buf] == NUL
      (if (i32.eqz (i32.load8_u (local.get $buf)))
        (then ;; reached the end of the value

          ;; value = buf - len
          (local.set $value (i32.sub (local.get $buf) (local.get $len)))

          ;; print_header_field(name, value, buf_console)
          (call $print_header_field
            (local.get $name) (local.get $name_len)
            (local.get $value) (local.get $len)
            (local.get $buf_console))

          (local.set $buf (i32.add (local.get $buf) (i32.const 1))) ;; buf++
          (local.set $len (i32.const 0))) ;; len = 0
        (else
          (local.set $len (i32.add (local.get $len) (i32.const 1))) ;; len++
          (local.set $buf (i32.add (local.get $buf) (i32.const 1))))) ;; buf++

      (local.set $values_len (i32.sub (local.get $values_len) (i32.const 1))) ;; values_len--

      ;; if values_len > 0 { continue } else { break }
      (br_if $values (i32.gt_u (local.get $values_len) (i32.const 0)))))

  ;; print_header_field prints a header field to the console, formatted like
  ;; "name: value".
  ;;
  ;; Note: This doesn't enforce a buf_limit as the runtime will panic on OOM.
  (func $print_header_field
    (param  $name i32) (param  $name_len i32)
    (param $value i32) (param $value_len i32)
    (param   $buf i32)

    (local $buf_0 i32)

    ;; buf_0 = buf
    (local.set $buf_0 (local.get $buf))

    ;; copy(mem[buf:], mem[name:name_len])
    (memory.copy
      (local.get $buf)
      (local.get $name)
      (local.get $name_len))

    ;; buf = buf + name_len
    (local.set $buf (i32.add (local.get $buf) (local.get $name_len)))

    ;; mem[buf++] = ':'
    (i32.store8 (local.get $buf) (i32.const (; ':'== ;) 58))
    (local.set $buf (i32.add (local.get $buf) (i32.const 1)))

    ;; mem[buf++] = ' '
    (i32.store8 (local.get $buf) (i32.const (; ' '== ;) 32))
    (local.set $buf (i32.add (local.get $buf) (i32.const 1)))

    ;; copy(mem[buf:], mem[value:value_len])
    (memory.copy
      (local.get $buf)
      (local.get $value)
      (local.get $value_len))

    ;; buf = buf + value_len
    (local.set $buf (i32.add (local.get $buf) (local.get $value_len)))

    ;; println(mem[buf_0:(buf - buf_0)])
    (call $println
      (local.get $buf_0)
      (i32.sub (local.get $buf) (local.get $buf_0))))

  ;; print_body prints the body of the given $kind to the console.
  (func $print_body (param $kind i32)
    (local $result i64)
    (local $len i32)

    ;; result = read_body(kind, heap_start, buf_limit)
    (local.set $result
      (call $read_body
        (local.get $kind)
        (global.get $heap_start)
        (global.get $buf_limit)))

    ;; len = uint32(result)
    (local.set $len (i32.wrap_i64 (local.get $result)))

    ;; don't console if there was no body
    (if (i32.eqz (local.get $len)) (then (return)))

    ;; if result & eof != eof { panic }
    (if (i64.ne
          (i64.and (local.get $result) (global.get $eof))
          (global.get $eof))
      (then unreachable)) ;; fail as we couldn't buffer the whole response.

    ;; println("")
    (call $println (global.get $heap_start) (i32.const 0))
    ;; println(mem[heap_start:len])
    (call $println (global.get $heap_start) (local.get $len)))

  (func $store_space (param $buf i32) (result i32)
    (i32.store8 (local.get $buf) (i32.const (; ' '== ;) 32))
    (i32.add (local.get $buf) (i32.const 1)))

  (func $store_status_code (param $buf i32) (param $status_code i32)
    (local $rem i32)

    ;; if status_code < 100 || status_code >> 599 { panic }
    (if (i32.or
          (i32.lt_u (local.get $status_code) (i32.const 100))
          (i32.gt_u (local.get $status_code) (i32.const 599)))
       (then (unreachable)))

    ;; write the 3 digits backwards, from right to left.
    (local.set $buf (i32.add (local.get $buf) (i32.const 3))) ;; buf += 3

    (loop $status_code_ne_zero
      ;; rem = status_code % 10
      (local.set $rem (i32.rem_u (local.get $status_code) (i32.const 10)))

      ;; mem[--buf] = rem + '0'
      (local.set $buf (i32.sub (local.get $buf) (i32.const 1)))
      (i32.store8
        (local.get $buf)
        (i32.add(local.get $rem) (i32.const (; '0'== ;) 48)))

      ;; status_code /= 10
      (local.set $status_code (i32.div_u (local.get $status_code) (i32.const 10)))

      ;; if $status_code != 0 { continue } else { break }
      (br_if $status_code_ne_zero (i32.ne (local.get $status_code) (i32.const 0)))))

  ;; println prints the input to the console with a newline
  (func $println (param $buf i32) (param $buf_len i32)
    (local $iovs i32) (local $iovs_len i32)

    (if (i32.eqz (local.get $buf_len))
      (then ;; write the predefined second I/O vector
        (local.set $iovs
          (i32.add (global.get $iovs) (i32.const 8))) ;; skip first iovec
        (local.set $iovs_len (i32.const 1)))
      (else ;; overwrite the first 8-byte I/O vector with $buf, $buf_len
        (i32.store (global.get $iovs) (local.get $buf))
        (i32.store
          (i32.add (global.get $iovs) (i32.const 4))
          (local.get $buf_len))
        (local.set $iovs (global.get $iovs))
        (local.set $iovs_len (i32.const 2))))

    ;; call fd_write and ensure there's no error code.
    (if (i32.ne
       (call $fd_write
         (i32.const 1) ;; STDOUT
         (local.get  $iovs)
         (local.get  $iovs_len)
         (global.get $ignored)) ;; write byte count to an ignored position.
       (i32.const 0)) ;; errnosuccess
     (then unreachable))) ;; fail as we couldn't write to STDOUT
)
