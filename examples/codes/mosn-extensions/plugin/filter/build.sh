function make_checker {
	go build -o checker checker.go
}

function make_mosn {
	mkdir ./build_mosn
	cp ../../../../../cmd/mosn/main/* ./build_mosn
	cp ./checker_filter.go ./build_mosn
	cd ./build_mosn
	go build -o mosn
	mv mosn ../
	cd ../
	rm -rf ./build_mosn
}

make_checker
make_mosn
