## Use MOSN to authorize JWT

## Introduction

+ This sample project demonstrates how to configure MOSN to authorize based on key-auth

## Preparation

+ A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/filter/keyauth/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```


## Catelog

```
main        // compiled MOSN
server.go   // Http Server
config.json // keyauth filter configuration
```

## Operation instructions

### Start HTTP Server 

```
go run server.go
```

### Start MOSN

```
./main start -c config.json
```

### Use CURL for verification

#### Verify that a request with a valid key is allowed

```
curl http://127.0.0.1:10345/prefix -s -o /dev/null  -H "mosnKey: key1" -w "%{http_code}\n" 
200

curl http://127.0.0.1:10345/path -s -o /dev/null  -H "mosnKey: key2" -w "%{http_code}\n" 
200
```

#### Verify that a request with an invalid key is denied

```
curl http://127.0.0.1:10345/prefix -s -o /dev/null  -H "mosnKey: key2" -w "%{http_code}\n" 
401

curl http://127.0.0.1:10345/path -s -o /dev/null  -H "mosnKey: errorKey" -w "%{http_code}\n" 
401
```