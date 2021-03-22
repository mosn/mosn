#!/bin/bash

export VERSION_FILE=$MOSN_ROOT/VERSION

export MAJOR_VERSION=$(cat $VERSION_FILE)

export PATH=$PATH:$MOSN_ROOT/build/bundles/$MAJOR_VERSION/binary

mosn start -c /etc/wasm/config.json