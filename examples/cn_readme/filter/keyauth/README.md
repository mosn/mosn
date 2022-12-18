## 在 MOSN 里基于 JWT 授权

## 简介

+ 该样例工程演示了如何配置使得 MOSN 可以基于密钥授权

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/filter/keyauth/
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
config.json // 基于 keyauth filter 的配置
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

#### 验证使用有效 key 的请求被允许
```
curl http://127.0.0.1:10345/prefix -s -o /dev/null  -H "mosnKey: key1" -w "%{http_code}\n" 
200

curl http://127.0.0.1:10345/path -s -o /dev/null  -H "mosnKey: key2" -w "%{http_code}\n" 
200
```

#### 验证使用无效 key 的请求被拒绝

```
curl http://127.0.0.1:10345/prefix -s -o /dev/null  -H "mosnKey: key2" -w "%{http_code}\n" 
401

curl http://127.0.0.1:10345/path -s -o /dev/null  -H "mosnKey: errorKey" -w "%{http_code}\n" 
401
```