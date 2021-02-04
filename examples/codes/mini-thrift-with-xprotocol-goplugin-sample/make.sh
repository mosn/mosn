#!/bin/bash

echo '==> Making goplugin for thrift protocol...'
cd ./thrift/codec && ./make.sh

cd -

echo '==> Making thrift demo app...'
cd ./thrift/tutorial && ./make.sh

cd -

echo '==> Making mosn...'
cd mosn && ./make.sh
