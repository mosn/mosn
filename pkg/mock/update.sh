#!/bin/bash

# source file => output file
files=(
../../vendor/mosn.io/api/network.go network.go
../../vendor/mosn.io/api/network_filter.go network_filter.go
../../vendor/mosn.io/api/stream_filter.go stream_filter.go
../../vendor/mosn.io/api/route.go api_route.go
../../vendor/mosn.io/api/header.go header.go
../../vendor/mosn.io/api/trace.go trace.go
../types/route.go mosn_route.go
../stagemanager/stage_manager.go stm_app.go
../types/stream.go stream.go
../types/tls.go tls.go
../types/upstream.go upstream.go
../mtls/tls_context.go tls_context.go
../mtls/types.go mtls_types.go
)

i=0
while [ $i -lt ${#files[@]} ]
do
    infile=${files[$i]}
    let i++
    outfile=${files[$i]}
    let i++

    if [ ! -f $infile ]; then
        echo "ERROR: $infile not existing"
        exit 1
    fi

    # echo $infile
    # echo $outfile
    mockgen -source=$infile -package=mock > $outfile || exit 1
done

# wasm.go
mockgen --build_flags=--mod=mod -package mock mosn.io/mosn/pkg/types ABIHandler,ABI,WasmFunction,WasmInstance,WasmModule,WasmVM,WasmPlugin,WasmPluginHandler,WasmPluginWrapper,WasmManager > wasm.go || exit 1
