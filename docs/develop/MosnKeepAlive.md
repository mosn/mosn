# MOSN 保活

## 使用 supervisor 保活

+ 使用 supervisor 监控程序的运行，在 MSON 挂掉的时候自动拉起，延迟时间从日志上看大概在 1-2s，
  其中 supervisor 中针对 MOSN 的配置可设置为（供参考）：
  
  ```bash
    command = /home/admin/mosn/bin/mosnd start -c /home/admin/mosn/conf/mosn.conf
    autostart=true       //mosn随supervisor一起启动
    autorestart=true     //设置mosn退出后自动启动
    startsecs=1          //启动成功时间默认1s
    user=admin
    stdout_logfile = /home/admin//usercenter_stdout.log
  ```
## MOSN 持久化服务订阅等信息的方法
通过将 Kubernetes、pilot等推送的“服务注册等服务信息”，回写到 MOSN 的配置文件即 `mosn.conf` 中进行持久化，在 MOSN 重启的时候，通过解析这两部分的配置来重新从配置文件拉取；

