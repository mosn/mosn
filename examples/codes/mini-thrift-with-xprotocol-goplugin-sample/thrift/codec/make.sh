go build -gcflags "all=-N -l" -buildmode=plugin -o thrift.so  \
  ./api.go \
  ./buffer.go \
  ./command.go \
  ./decoder.go \
  ./encoder.go \
  ./mapping.go \
  ./matcher.go \
  ./protocol.go \
  ./types.go \
  ./logger.go
