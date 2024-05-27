FROM golang:1.18

ENV GO111MODULE=on

ENV MOSN_ROOT /go/src/mosn.io/mosn

COPY ./build    $MOSN_ROOT/build
COPY ./cmd      $MOSN_ROOT/cmd
COPY ./configs  $MOSN_ROOT/configs/
COPY ./pkg      $MOSN_ROOT/pkg
COPY ./istio    $MOSN_ROOT/istio
COPY ./VERSION  $MOSN_ROOT/VERSION
COPY ./Makefile $MOSN_ROOT/Makefile
COPY ./go.mod   $MOSN_ROOT/go.mod
COPY ./go.sum   $MOSN_ROOT/go.sum

COPY ./configs/mosn_config_wasm.json    /etc/wasm/config.json
COPY ./etc/wasm/filter.wasm             /etc/wasm/filter.wasm

WORKDIR $MOSN_ROOT

RUN export GOPROXY=https://goproxy.cn \
    && make build-local tags=wasmer \
    && chmod 777 $MOSN_ROOT/build/contrib/builder/wasm/bootstrap.sh \
    && rm -rf $MOSN_ROOT/cmd \
    && rm -rf $MOSN_ROOT/pkg

ENTRYPOINT ["/bin/bash", "-c", "/go/src/mosn.io/mosn/build/contrib/builder/wasm/bootstrap.sh"]
