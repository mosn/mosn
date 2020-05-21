## MOSN的扩展机制示例说明

### 如何运行示例

+ 切换到对应示例的目录下，执行build.sh
+ 运行MOSN 
```
cd ${target}
bash build.sh
./mosn start -c config.json
```

### simple_streamfilter

+ 一个简单的stream filter示例
+ 该stream filter实现了如下功能：
  + 只允许请求的Header中包含Filter配置中所有的Header信息时，MOSN才会转发对应的请求，否则直接返回403错误(以HTTP为例)
+ 示例默认配置，要求Header中必须包含`User:admin`
```
// 返回403
curl -i http://127.0.0.1:2046/
// 返回502, 因为没有启动后端Server
curl -H"User:admin" -i http://127.0.0.1:2046/
// 监听一个8080端口的HTTP Server，如examples/codes/http-sample下的server.go
// 返回200
curl -H"User:admin" -i http://127.0.0.1:2046/
```

### plugin/load_cert

+ 一个简单的多进程示例
+ 使用Go语言编译了一个独立的进程，该进程的功能用于返回TLS证书信息,模拟一个证书管理加密的进程
+ MOSN启动以后，可以发现正确加载了来自独立进程输出的证书信息
```
// 明文方式访问，tls握手失败
curl -i http://127.0.0.1:2046/
// HTTPS 方式访问，因为不识别证书，会返回证书校验错误
curl https:///127.0.0.1:2046/
```

### plugin/filter

+ 结合多进程与stream filter的简单示例
  + 独立进程实现了同`simple_streamfilter`相同的逻辑，其配置信息来自启动参数
  + stream filter通过多进程框架加载实现逻辑，完成对应的逻辑
+ 验证方式参考`simple_streamfilter`


### plugin/so

+ 一个MOSN加载SO的示例
+ 实现一个MOSN的"通用"能力，可通过so的方式动态加载任意stream filter的实现
+ so的实现来自`simple_streamfilter`，不过存在一些区别（so规范)
  + Factory函数名字一定叫`CreateFilterFactory`
  + 不需要`init()`注册
+ 验证方式参考`simple_streamfilter`
