## 在 MOSN 设置ip_access filter

## 简介

+ 该样例工程演示了如何配置使得 MOSN 设置ip_access filter
+ 默认情况下，允许所有ip访问，启发于[ngx_http_access_module](http://nginx.org/en/docs/http/ngx_http_access_module.html)

## 配置

```
{
  "stream_filters": [
    {
      "type": "ip_access",  // filter name
      "config": {
       "default_action": "allow", // deny or allow ,default is allow， lowest priority takes effect
        "header": "",  // trusted header to get real ip,if null will get remote addr
        "ips": [
          {
            "action": "deny", // deny or allow
            "addrs": [
              "127.0.0.1" // ip or subnet,0.0.0.0/0  will match all ip ,just like nginx deny all | allow all
            ]
          }
        ]
      }
    }
  ]
}
```

下面的代码和nginx对比

样例1

```json

{
  "type": "ip_access",
  "config": {
    "header": "",
    "default_action": "allow",
    "ips": [
      {
        "action": "deny",
        "addrs": [
          "127.0.0.1"
        ]
      }
    ]
  }
}
```

等价于 nginx

```conf
deny 127.0.0.1
allow all
```

样例2

```json
{
  "type": "ip_access",
  "config": {
    "header": "",
    "default_action": "deny",
    "ips": [
      {
        "action": "deny",
        "addrs": [
          "192.168.1.1"
        ]
      },
      {
        "action": "allow",
        "addrs": [
          "192.168.1.0/24",
          "10.1.1.0/16",
          "2001:0db8::/32"
        ]
      }
    ]
  }
}
```

等价于 nginx

```
deny  192.168.1.1;
allow 192.168.1.0/24;
allow 10.1.1.0/16;
allow 2001:0db8::/32;
deny  all;

```

## 准备

需要一个编译好的MOSN程序

```

cd ${projectpath}/cmd/mosn/main go build

```

+ 示例代码目录

```

${targetpath} = ${projectpath}/examples/codes/filter/ipaccess/

```

+ 将编译好的程序移动到示例代码目录

```

mv main ${targetpath}/ cd ${targetpath}

```

## 目录结构

```

main // 编译完成的MOSN程序 server.go // 模拟的Http Server config.json // 基于 filter ip_access 的配置

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

#### 验证非授权ip无法访问

ip 填写非授权ip

```

curl http://127.0.0.1:2046 -s -o /dev/null -H "X-FORWARD-IP: 127.0.0.1"
403

```

#### 验证授权可以访问

```

curl http://{ip}:2046 -s -o /dev/null -H "X-FORWARD-IP: {ip}"
200

```

