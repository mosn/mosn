# Notice

please init your zk first!

## Running steps

0. start mosn with this example's config

```shell
go build -tags dubbo
```

```shell
./mosn start
```

1. go to server dir, and run:

```shell
make run
```


2. use httpie to publish service:

```shell
http --json post localhost:22222/pub registry:='{"type":"zookeeper", "addr" : "127.0.0.1:2181"}' service:='{"interface" : "com.ikurento.user.UserProvider", "methods" :["GetUser"], "port" : "20000", "group" : "", "version" : ""}' --verbose
```

3. go to client dir, and run:

```shell
make run
```

