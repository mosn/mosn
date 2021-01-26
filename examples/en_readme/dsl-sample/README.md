## Use DSL to control the MOSN request processing 

## Introduction

+ The sample project demonstrates how to configure a DSL filter to process requests.
 
## Preparation

+ A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/dsl-sample/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```

## Catelog

```
main        // compiled MOSN
config.json // Configure of MOSN

```

## Operation instructions

### Start MOSN

+ Use config.json to run MOSN.

```
./main start -c config.json
```


### Use CURL for verification

#### Scenario 1

```
curl localhost:2046 -v -H "host:dslhost"
```

The output should be:

```text
* About to connect() to localhost port 2046 (#0)
*   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 2046 (#0)
> GET / HTTP/1.1
> User-Agent: curl/7.29.0
> Accept: */*
> host:dslhost
> 
< HTTP/1.1 200 OK
< Date: Mon, 20 Jul 2020 15:55:06 GMT
< Content-Type: text/plain; charset=utf-8
< Content-Length: 10
< Host: dslhost
< User-Agent: curl/7.29.0
< Accept: */*
< Dsl: dsl
< Upstream_addr: localhost
< 
* Connection #0 to host localhost left intact
hello DSL!

```

#### Scenario 2

```
curl localhost:2046 -v
```

The output should be:

```text
* About to connect() to localhost port 2046 (#0)
*   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 2046 (#0)
> GET / HTTP/1.1
> User-Agent: curl/7.29.0
> Host: localhost:2046
> Accept: */*
> 
< HTTP/1.1 404 Not Found
< Date: Mon, 20 Jul 2020 15:55:12 GMT
< Content-Length: 0
< Host: localhost:2046
< User-Agent: curl/7.29.0
< Accept: */*
< Dsl: dsl
< Upstream_addr: localhost
< 
* Connection #0 to host localhost left intact
```

