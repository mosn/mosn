# 0.2.0 TLS 扩展
## 扩展的几个函数
+ 获取tls.Certificate的扩展，golang原生只支持明文的证书/私钥方式获取
    + 由扩展返回tls.Certificate，并且返回的tls.Certificate中PrivateKey也可以扩展，可以满足keyless的需求
+ VerifyPeerCertificate函数的扩展，该函数是golang的tls.Config支持的扩展方式，可以修改TLS证书校验的方式
+ 获取可信任的CA

## 改造结果
+ 新增扩展，扩展的Factory，tls在使用的时候，根据注册的type获取factory，再根据配置生成对应的扩展提供tls证书
+ 相关的配置文件、配置文件解析存在修改
+ context 和 contextManager修改  
+ tls.Config对于扩展的处理