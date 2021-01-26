
NAME=mosn/py
VERSION= 1.0

build:  
	rm -rf ./build
	docker run --rm -v `pwd`:/work -w /work ${NAME}:${VERSION} /bin/bash -c "/usr/local/python3/bin/pyinstaller -F plugin.py" 

build-image:
	echo starting build docker image...
	docker build -t ${NAME}:${VERSION}  .

