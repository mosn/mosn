function make_mosn {
	mkdir ./build_mosn
	cp ../../../../../cmd/mosn/main/* ./build_mosn
	cd ./build_mosn
	go build -o mosn
	mv mosn ../
	cd ../
	rm -rf ./build_mosn
}

function make_so {
	go build -buildmode=plugin ./simple_so.go
}

make_so
make_mosn
