配置示例：
```json
{
    "servers": [
        {
            "global_log_roller": "size=100 age=10 keep=10 compress=off",
            "default_log_path": "/home/admin/mosn/logs/default.log",
            "default_log_level": "ERROR",
            "listeners": [
                {
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
* global_log_roller
  全局的日志轮转参数，针对所有日志生效，包括accesslog，defaultlog等。
  * size 表示日志达到多少M进行轮转，单位： M
  * age 表示最大保存多少天内的日志
  * keep 表示最大保存多少个日志
  * compress 表示是否压缩（on/off)
  `"global_log_roller": "size=100 age=10 keep=10 compress=off"`

* default_log_path
  默认的错误日志路径

* default_log_level
  默认的错误日志等级
  * ERROR
  * WARN
  * INFO
  * DEBUG
  * TRACE

* access_logs
  请求日志
  * log_path 日志路径
  * log_format 日志格式

注意事项：
* 默认配置为按天轮转。
* 日志按时间轮转优先级最高，配置了之后其他规F则都失效。