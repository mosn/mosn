
```json
{
    "servers": [
        {
            "default_log_path": "/home/admin/mosn/logs/default.log",
            "default_log_level": "ERROR",
            "default_log_roller": "size=100 age=10 keep=10 compress=off",
            "listeners": [
                {
                    "log_path": "/home/admin/mosn/logs/egress.log",
                    "log_level": "ERROR",
                    "access_logs": [
                        {
                            "log_path": "/home/admin/mosn/logs/access.log",
                            "log_format": "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %REQ.requestid% %REQ.cmdcode% %RESP.requestid% %RESP.service%"
                        }
                    ]
                }
            ]
        }
    ]
}
```
* default_log_path
  默认的错误日志路径

* default_log_level
  默认的错误日志等级
  * ERROR
  * WARN
  * INFO
  * DEBUG
  * TRACE

* default_log_roller
  默认的日志轮转参数
  * size 表示日志达到多少M进行轮转，单位： M
  * age 表示最大保存多少天内的日志
  * keep 表示最大保存多少个日志
  * compress 表示是否压缩（on/off)
  `"default_log_roller": "size=100 age=10 keep=10 compress=off"`

* log_path
  listener级别日志路径

* log_level
  listener级别日志等级

* access_logs
  请求日志
  * log_path 日志路径
  * log_format 日志格式

default_log_roller只有全局配置，不支持listener级别。
如果不配置，日志默认按天轮转