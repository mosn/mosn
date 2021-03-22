FROM golang:1.14.13

# use go mod vendor mode for build and test
ENV GO111MODULE=off

ENV MOSN_ROOT /go/src/mosn.io/mosn

COPY ./build    $MOSN_ROOT/build
COPY ./cmd      $MOSN_ROOT/cmd
COPY ./configs  $MOSN_ROOT/configs/
COPY ./pkg      $MOSN_ROOT/pkg
COPY ./vendor   $MOSN_ROOT/vendor
COPY ./etc      $MOSN_ROOT/etc
COPY ./VERSION  $MOSN_ROOT/VERSION
COPY ./Makefile $MOSN_ROOT/Makefile
COPY ./go.mod   $MOSN_ROOT/go.mod
COPY ./go.sum   $MOSN_ROOT/go.sum

WORKDIR $MOSN_ROOT

RUN make build-local \
    && chmod 777 $MOSN_ROOT/build/contrib/builder/wasm/bootstrap.sh \
    && rm -rf $MOSN_ROOT/cmd \
    && rm -rf $MOSN_ROOT/pkg

ENTRYPOINT ["/bin/bash", "-c", "/go/src/mosn.io/mosn/build/contrib/builder/wasm/bootstrap.sh"]