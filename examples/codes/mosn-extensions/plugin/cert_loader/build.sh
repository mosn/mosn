function make_build_mosn {
	mkdir ./build_mosn
	cp ../../../../../cmd/mosn/main/* ./build_mosn
	cp ./mosn_ext.go ./build_mosn
	cd ./build_mosn
	go build -o mosn
	mv mosn ../
	cd ../
	rm -rf ./build_mosn
}

function make_loader {
	go build -o loader loader.go
}

make_loader
make_build_mosn
