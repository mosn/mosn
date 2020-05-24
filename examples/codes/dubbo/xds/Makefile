IMAGE_REPO 	 := "registry.cn-hangzhou.aliyuncs.com/sink-demo"
MOSN_VERSION := "v0.0.2"
VERSION 	 := "v0.0.1"

build-consumer:
	docker build -t ${IMAGE_REPO}/dubbo-consumer:${VERSION} -f ./consumer/Dockerfile .

build-provider:
	docker build -t ${IMAGE_REPO}/dubbo-provider:${VERSION} -f ./provider/Dockerfile .

build-mosn:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./mosn/mosn ../../../../cmd/mosn/main
	docker build -t ${IMAGE_REPO}/mosn:${MOSN_VERSION} -f ./mosn/Dockerfile .

push:
	docker push ${IMAGE_REPO}/dubbo-consumer:${VERSION}
	docker push ${IMAGE_REPO}/dubbo-provider:${VERSION}
	docker push ${IMAGE_REPO}/mosn:${MOSN_VERSION}

push-mosn: build-mosn
	docker push ${IMAGE_REPO}/mosn:${MOSN_VERSION}	

build-push: build-consumer build-provider build-mosn push
