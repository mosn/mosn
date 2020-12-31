## Use MOSN to authorize JWT

## Introduction

+ This sample project demonstrates how to configure MOSN to authorize based on JWT

## Preparation

+ A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/http-sample/
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
config.json // jwt filter configuration
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

#### Verify that a request with an invalid JWT is denied

```
curl http://127.0.0.1:2046 -s -o /dev/null  -H "Authorization: Bearer token" -w "%{http_code}\n" 
401
```

#### Verify that a request without a JWT is allowed because there is no authorization policy

```
curl http://127.0.0.1:2046 -s -o /dev/null  -w "%{http_code}\n" 
200
```

#### Verify that a request with a valid JWT is allowed

```
TOKEN=eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjQ2ODU5ODk3MDAsImZvbyI6ImJhciIsImlhdCI6MTUzMjM4OTcwMCwiaXNzIjoidGVzdGluZ0BzZWN1cmUuaXN0aW8uaW8iLCJzdWIiOiJ0ZXN0aW5nQHNlY3VyZS5pc3Rpby5pbyJ9.CfNnxWP2tcnR9q0vxyxweaF3ovQYHYZl82hAUsn21bwQd9zP7c-LS9qd_vpdLG4Tn1A15NxfCjp5f7QNBUo-KC9PJqYpgGbaXhaGx7bEdFWjcwv3nZzvc7M__ZpaCERdwU7igUmJqYGBYQ51vr2njU9ZimyKkfDe3axcyiBZde7G6dabliUosJvvKOPcKIWPccCgefSj_GNfwIip3-SsFdlR7BtbVUcqR-yv-XOxJ3Uc1MI0tz3uMiiZcyPV7sNCU4KRnemRIMHVOfuvHsU60_GhGbiSFzgPTAa9WTltbnarTbxudb_YEOx12JiwYToeX0DCPb43W1tzIBxgm8NxUg
curl http://127.0.0.1:2046 -s -o /dev/null -H "Authorization: Bearer ${TOKEN}"  -w "%{http_code}\n"
200
```