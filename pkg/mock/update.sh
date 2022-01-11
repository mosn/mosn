#!/bin/bash

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
../types/upstream.go upstream.go
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
    echo $infile
    echo $outfile
    mockgen -source=$infile -package=mock > $outfile
done
