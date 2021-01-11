# build go and mv to current path

GO111MODULE=off go build -buildmode=plugin pluginsource/*.go
echo "run test cases"
GO111MODULE=off go test -tags MOSNTest -failfast -v -p 1 *.go
code=$?
if [[ $code -eq 0 ]]; then
	echo "all cases success"
else 
	echo "----FAILED---"
fi
rm -rf *.so
exit $code
