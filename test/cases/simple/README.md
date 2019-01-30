This is the simple test cases, contains two part:

+ client - mosn - server
  + sofarpc(boltv1) (boltv1.go)
  + http1 (http1.go)
  + http2 (http2.go)

+ client - mosn - mosn - server
  + http1 - http1 - http1 - http1 (http1.go)
  + http1 - http2 - http2 - http1 (http1_convert.go)
  + http2 - http1 - http1 - http2 (http2_convert.go)
  + http2 - http2 - http2 - http2 (http2.go)
  + sofarpc(boltv1) - http1 - http1 - sofarpc(boltv1) (boltv1_convert.go)
  + sofarpc(boltv1) - http2 - http2 - sofarpc(boltv1) (boltv1_convert.go)
  + sofarpc(boltv1) - sofarpc - sofarpc - sofarpc(boltv1) (boltv1.go)

These cases just verify client received what server responsed, mosn will not modify any thing
