;; This example module is written in WebAssembly Text Format to show the
;; how a handler works and that it is decoupled from other ABI such as WASI.
;; Most users will prefer a higher-level language such as C, Rust or TinyGo.
(module $router
  ;; get_uri writes the URI to memory if it isn't larger than $buf_limit. The
  ;; result is its length in bytes.
  (import "http_handler" "get_uri" (func $get_uri
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; set_uri overwrites the URI with one read from memory.
  (import "http_handler" "set_uri" (func $set_uri
    (param $uri i32) (param $uri_len i32)))

  ;; set_header_value overwrites a header of the given $kind and $name with a
  ;; single value.
  (import "http_handler" "set_header_value" (func $set_header_value
    (param $kind i32)
    (param $name i32) (param $name_len i32)
    (param $value i32) (param $value_len i32)))

  ;; write_body reads $buf_len bytes at memory offset `buf` and writes them to
  ;; the pending $kind body.
  (import "http_handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  ;; http_handler guests are required to export "memory", so that imported
  ;; functions like "log" can read memory.
  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; uri is an arbitrary area to write data.
  (global $uri       i32 (i32.const 1024))
  (global $uri_limit i32 (i32.const  256))

  (global $path_prefix i32 (i32.const 0))
  (data (i32.const 0) "/host")
  (global $path_prefix_len i32 (i32.const 5))

  (global $content_type_name i32 (i32.const 32))
  (data (i32.const 32) "Content-Type")
  (global $content_type_name_len i32 (i32.const 12))

  (global $content_type_value i32 (i32.const 64))
  (data (i32.const 64) "text/plain")
  (global $content_type_value_len i32 (i32.const 10))

  (global $body i32 (i32.const 96))
  (data (i32.const 96) "hello world")
  (global $body_len i32 (i32.const 11))

  ;; handle_request implements a simple HTTP router.
  (func (export "handle_request") (result (; ctx_next ;) i64)

    (local $uri_len i32)

    ;; First, read the uri into memory if not larger than our limit.

    ;; uri_len = get_uri(uri, uri_limit)
    (local.set $uri_len
      (call $get_uri (global.get $uri) (global.get $uri_limit)))

    ;; if uri_len > uri_limit { next() }
    (if (i32.gt_u (local.get $uri_len) (global.get $uri_limit))
      (then
        (return (i64.const 1)))) ;; dispatch if the uri is too long.

    ;; Next, strip any paths starting with '/host' and dispatch.

    ;; if path_prefix_len <= uri_len
    (if (i32.eqz (i32.gt_u (global.get $path_prefix_len) (local.get $uri_len)))
      (then

        (if (call $memeq ;; uri[0:path_prefix_len] == path_prefix
              (global.get $uri)
              (global.get $path_prefix)
              (global.get $path_prefix_len))
          (then
            (call $set_uri ;; uri = uri[path_prefix_len:]
              (i32.add (global.get $uri)     (global.get $path_prefix_len))
              (i32.sub (local.get  $uri_len) (global.get $path_prefix_len)))
            (return (i64.const 1)))))) ;; dispatch with the stripped path.

    ;; Otherwise, serve a static response.
    (call $set_header_value
      (i32.const 1) ;; header_kind_response
      (global.get $content_type_name)
      (global.get $content_type_name_len)
      (global.get $content_type_value)
      (global.get $content_type_value_len))
    (call $write_body
      (i32.const 1) ;; body_kind_response
      (global.get $body)
      (global.get $body_len))
    (return (i64.const 0))) ;; don't call the next handler

  ;; handle_response is no-op as this is a request-only handler.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32))

  ;; memeq is like memcmp except it returns 0 (ne) or 1 (eq)
  (func $memeq (param $ptr1 i32) (param $ptr2 i32) (param $len i32) (result i32)
    (local $i1 i32)
    (local $i2 i32)
    (local.set $i1 (local.get $ptr1)) ;; i1 := ptr1
    (local.set $i2 (local.get $ptr2)) ;; i2 := ptr1

    (loop $len_gt_zero
      ;; if mem[i1] != mem[i2]
      (if (i32.ne (i32.load8_u (local.get $i1)) (i32.load8_u (local.get $i2)))
        (then (return (i32.const 0)))) ;; return 0

      (local.set $i1  (i32.add (local.get $i1)  (i32.const 1))) ;; i1++
      (local.set $i2  (i32.add (local.get $i2)  (i32.const 1))) ;; i2++
      (local.set $len (i32.sub (local.get $len) (i32.const 1))) ;; $len--

      ;; if $len > 0 { continue } else { break }
      (br_if $len_gt_zero (i32.gt_u (local.get $len) (i32.const 0))))

    (i32.const 1)) ;; return 1
)
