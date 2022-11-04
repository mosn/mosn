(module $protocol_version

  (import "http_handler" "get_protocol_version" (func $get_protocol_version
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  (import "http_handler" "write_body" (func $write_body
    (param $kind i32)
    (param $buf i32) (param $buf_len i32)))

  (memory (export "memory") 1 1 (; 1 page==64KB ;))

  ;; handle_request writes the protocol version to the response body.
  (func (export "handle_request") (result (; ctx_next ;) i64)
    (local $len i32)

    ;; read the protocol version into memory at offset zero.
    (local.set $len
      (call $get_protocol_version (i32.const 0) (i32.const 1024)))

    ;; write the protocol version to the response body.
    (call $write_body
      (i32.const 1) ;; body_kind_response
      (i32.const 0) (local.get $len))

    ;; skip any next handler as we wrote the response body.
    (return (i64.const 0)))

  ;; handle_response is no-op as this is a request-only handler.
  (func (export "handle_response") (param $reqCtx i32) (param $is_error i32))
)
