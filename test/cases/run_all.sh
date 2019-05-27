# build go and mv to current path
function makebuild { 
	OPWD=$(pwd)
	cd ../../cmd/mosn/main
	go build
	CPWD=$(pwd)
	mv ./main "$OPWD/test_mosn"
	cd $OPWD
}
# traverse cases files
function traverse {
	for f in `find ./`
	do
		FILE=$(echo $f | awk -F "/" '{print $NF}')
		EXT="${FILE#*.}"
		if [ "go" == "$EXT" ]; then
			"$1" $f
			sleep 1
		fi
	done
}
# run case
function run {
	go run "$1" -m=./test_mosn
	code=$?
	if [[ $code -ne 0 ]]; then
		rm -f ./test_mosn
		exit 1
	fi
}
echo "build mosn binary"
makebuild
echo "run test cases"
traverse run
echo "pass all test cases"
rm -f ./test_mosn
