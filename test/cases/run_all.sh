# build go and mv to current path
function makebuild { 
	OPWD=$(pwd)
	mkdir -p $OPWD/tmpmain
	# copy test code for extend
	cp -R $OPWD/extends/* $OPWD/tmpmain
	# copy x_example protocol codec
	cp -R ../../examples/codes/xprotocol_with_goplugin_example/codec* tmpmain/pluginsource
	# copy mosn main code
	cp ../../cmd/mosn/main/* $OPWD/tmpmain
	# GO BUILD
	cd $OPWD/tmpmain
	go build -tags=mosn_debug -o main
	mv ./main "$OPWD/test_mosn"
	# compile so file
	cd $OPWD
	ls tmpmain/pluginsource/*.go | xargs -I % sh -c 'go build -buildmode=plugin %'
	rm -rf $OPWD/tmpmain
	echo $OPWD/test_mosn
}

# run 
echo "build mosn binary"
bin=$(makebuild)
echo "run test cases"
GO111MODULE=on go test -tags MOSNTest -failfast -v -p 1 ./... -args -m=$bin
code=$?
rm -f ./test_mosn
if [[ $code -eq 0 ]]; then
	echo "all cases success"
else 
	echo "----FAILED---"
fi
rm -rf *.so
exit $code
