## Set ip_access filter in MOSN

## Introduction

+ This sample project demonstrates how to configure MOSN to set ip_access filter
+ By default, all ip access is allowed, inspired by [ngx_http_access_module](http://nginx.org/en/docs/http/ngx_http_access_module.html)

## Configuration

```
{
  "stream_filters": [
    {
      "type": "ip_access", // filter name
      "config": {
       "default_action": "allow", // deny | allow, lowest priority takes effect
        "header": "", // trusted header to get real ip,if null will get remote addr
        "ips": [
          {
            "action": "deny", // deny | allow  
            "addrs": [
              "127.0.0.1" // ip or subnet,0.0.0.0/0 will match all ip ,just like nginx deny all | allow all
            ]
          }
        ]
      }
    }
  ]
}
```

The following code is compared with nginx

Example 1

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

Equivalent to nginx

```conf
deny 127.0.0.1
allow all
```

Example 2

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

Equivalent to nginx

```
deny 192.168.1.1;
allow 192.168.1.0/24;
allow 10.1.1.0/16;
allow 2001:0db8::/32;
deny all;

```

## Prepare

Need a compiled MOSN program

```

cd ${projectpath}/cmd/mosn/main go build

```

+ Sample code directory

```

${targetpath} = ${projectpath}/examples/codes/filter/ipaccess/

```

+ Move the compiled program to the sample code directory

```

mv main ${targetpath}/ cd ${targetpath}

```

## Directory Structure

```

main // compiled MOSN program server.go // simulated Http Server config.json // configuration based on filter ip_access

```

## Operation instructions

### Start an HTTP Server

```

go run server.go

```

### Start MOSN

```

./main start -c config.json

```

### Use CURL to verify

#### Verify that unauthorized ip cannot be accessed

ip fill in unauthorized ip

```

curl http://127.0.0.1:2046 -s -o /dev/null -H "X-FORWARD-IP: 127.0.0.1"
403

```

#### Verify that authorization can access

```

curl http://{ip}:2046 -s -o /dev/null -H "X-FORWARD-IP: {ip}"
200

```