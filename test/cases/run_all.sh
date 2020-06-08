# build go and mv to current path
function makebuild { 
	OPWD=$(pwd)
	cd ../../cmd/mosn/main
	go build
	CPWD=$(pwd)
	mv ./main "$OPWD/test_mosn"
	echo $OPWD/test_mosn
}
echo "build mosn binary"
bin=$(makebuild)
echo "run test cases"
go test -tags MOSNTest -failfast -v -p 1 ./... -args -m=$bin
code=$?
rm -f ./test_mosn
exit $code
