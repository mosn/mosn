## 在 MOSN 里基于 JWT 授权

## 简介

+ 该样例工程演示了如何配置使得 MOSN 可以基于 JWT 授权

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/filter/jwtauthn/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的MOSN程序
server.go   // 模拟的Http Server
config.json // 基于 jwt filter 的配置
```

## 运行说明

### 启动一个HTTP Server

```
go run server.go
```

### 启动MOSN

```
./main start -c config.json
```

### 使用CURL进行验证

#### 验证使用无效 JWT 的请求被拒绝

```
curl http://127.0.0.1:2046 -s -o /dev/null  -H "Authorization: Bearer token" -w "%{http_code}\n" 
401
```

#### 验证没有 JWT 令牌的请求被允许，因为配置里的策略不包含授权策略

```
curl http://127.0.0.1:2046 -s -o /dev/null  -w "%{http_code}\n" 
200
```

#### 验证使用有效 JWT 的请求被允许

```
TOKEN=eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjQ2ODU5ODk3MDAsImZvbyI6ImJhciIsImlhdCI6MTUzMjM4OTcwMCwiaXNzIjoidGVzdGluZ0BzZWN1cmUuaXN0aW8uaW8iLCJzdWIiOiJ0ZXN0aW5nQHNlY3VyZS5pc3Rpby5pbyJ9.CfNnxWP2tcnR9q0vxyxweaF3ovQYHYZl82hAUsn21bwQd9zP7c-LS9qd_vpdLG4Tn1A15NxfCjp5f7QNBUo-KC9PJqYpgGbaXhaGx7bEdFWjcwv3nZzvc7M__ZpaCERdwU7igUmJqYGBYQ51vr2njU9ZimyKkfDe3axcyiBZde7G6dabliUosJvvKOPcKIWPccCgefSj_GNfwIip3-SsFdlR7BtbVUcqR-yv-XOxJ3Uc1MI0tz3uMiiZcyPV7sNCU4KRnemRIMHVOfuvHsU60_GhGbiSFzgPTAa9WTltbnarTbxudb_YEOx12JiwYToeX0DCPb43W1tzIBxgm8NxUg
curl http://127.0.0.1:2046 -s -o /dev/null -H "Authorization: Bearer ${TOKEN}"  -w "%{http_code}\n"
200
```