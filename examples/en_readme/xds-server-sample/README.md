## MOSN uses xds-server to achieve dynamic configuration

## Introduction

+ This sample project demonstrates how to configure MOSN to use a custom xds server for dynamic configuration
+ This sample project is only applicable to istio 1.10.0 and later, envoy v3

## Prepare

+ requires a compiled MOSN program

````
cd ${projectpath}/cmd/mosn/main
go build
````

+ sample code directory

````
${targetpath} = ${projectpath}/examples/codes/xds-server-sample/
````

+ Move the compiled program to the sample code directory

````
mv main ${targetpath}/
cd ${targetpath}
````


## Running instructions

### Start upstream server

````
   go run cmd/upstream/main.go

````

### Start xds-server


````
go run cmd/server/main.go

````

### Start mosn:

````
./main start -c config.json
````


## verify

Request upstream and request mosn return the same result
```bash
curl localhost:18080
curl localhost:10000
````

return

```bash
Default message
````

Modify confg/config.yaml, you can make mosn modify the corresponding behavior