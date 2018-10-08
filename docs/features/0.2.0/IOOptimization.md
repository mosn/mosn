# 0.2.0 Goroutine 优化

# 整体方案
- 底层使用epoll + 事件驱动编程模型，替代golang默认的epoll + 协程阻塞编程模型。
- IO协程池化，减少大量链接场景协程调度开销和内存开销。
- 开关配置，用户可根据自身的场景，决定使用何种协程模型。

# 细节变更
## epoll + 事件驱动编程模型
connection开启读写处理时，若当前模式为netpoll，则将自身的可读事件及其对应处理函数添加到全局的epoll wait的event loop中。

示例改造代码：
```go
//create poller
poller, err := netpoll.New(nil)
if err != nil {
	// handle error
}

// register read/write events instead of starting I/O goroutine
read, _ := netpoll.HandleReadOnce(c.RawConn())
poller.Start(read, func(e netpoll.Event) {
	        if e&netpoll.EventReadHup != 0 {
		    (*poller).Stop(read)
		     // process hup
                     ...
		    return
	        }

		// Read logic
	        go doRead()

		// Enable read event again
		poller.Resume(read)
	})
})
```

在链接关闭时，若此链接以netpoll模式运行，则需要将其注册的读写事件从event loop中注销掉。
示例代码：
```go
        // wait for io loops exit, ensure single thread operate streams on the connection
	if c.internalLoopStarted {
		// because close function must be called by one io loop thread, notify another loop here
		close(c.internalStopChan)
	} else if c.eventLoop != nil {
		c.eventLoop.Unregister(c.id)
	}
```


## IO协程池化
可读事件触发时，从协程池中获取一个goroutine来执行读处理，而不是新分配一个goroutine。以此来控制高并发下的协程数量。

示例改造代码：
```go
//create poller
poller, err := netpoll.New(nil)
if err != nil {
	// handle error
}

// register read/write events instead of starting I/O goroutine
read, _ := netpoll.HandleReadOnce(c.RawConn())
poller.Start(read, func(e netpoll.Event) {
	// No more calls will be made for conn until we call epoll.Resume().
	if e&netpoll.EventReadHup != 0 {
		(*poller).Stop(read)
		 // process hup
                 ...
		return
	}
	
	// Use worker pool to process read event
	pool.Schedule(func() {
		// Read logic
		doRead()

		// Enable read event again
		poller.Resume(read)
	})
})
```

需要注意的是，由于conn.write可能会阻塞协程，因此对于写操作的池化，我们采取了更为宽松的策略：池化的常驻协程数量仍然是固定的，但是允许新增临时协程。
池化代码：
```go
type SimplePool struct {
	work chan func()
	sem chan struct{}
}


func NewSimplePool(size int) *SimplePool {
	return &SimplePool{
		work: make(chan func()),
		sem:  make(chan struct{}, size),
	}
}

func (p *SimplePool) Schedule(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.worker(task)
	}
}

// for write schedule
func (p *SimplePool) ScheduleAlways(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.worker(task)
	default:
		go task()
	}
}

func (p *SimplePool) worker(task func()) {
	defer func() { <-p.sem }()
	for {
		task()
		task = <-p.work
	}
}
```
## 开关配置
新增一个全局配置来控制链接的读写协程运作模式:netpoll+协程池或者链接级别的读写协程。
示例代码：
```go
func (c *connection) Start(lctx context.Context) {
	c.startOnce.Do(func() {
		if UseNetpollMode {
			c.attachEventLoop(lctx)
		} else {
			c.startRWLoop(lctx)
		}
	})
}
```
示例配置：
```json
"servers": [
        {
 	  "use_netpoll_mode": true,
	}
]
```
# 性能对比
## goroutine数量

| 10,000 长连接 | 优化前 | 优化后 |
| -------- | -------- | -------- |
| goroutine  |  20040  | 40    |

## 内存占用
主要是goroutine stack的节省。

| 10,000 长连接 | 优化前 | 优化后 |
| -------- | -------- | -------- |
| stack     |   117M   | 1M     |

## 大量链接场景下的性能

> 10000长连接，300qps

|  | 优化前 | 优化后 |
| -------- | -------- | -------- |
| CPU     |  14%    | 8%     |

> 10000长连接，3000qps

|  | 优化前 | 优化后 |
| -------- | -------- | -------- |
| CPU     |    52%  |  40%   |

> 10000长连接，10000qps

|  | 优化前 | 优化后 |
| -------- | -------- | -------- |
| CPU     |   90%   |   91%   |

## 压测场景下的性能

| 指标 | IO协程 | netpoll |
| -------- | -------- | -------- |
| QPS     |   23000   |   22000   |
| CPU     |   100%   |   100%   |
| RT     |   9ms   |   10ms   |

## 分析

对于非活跃链接占据多数的场景，可以有效降低goroutine数量、内存占用以及CPU开销。但是对于活跃链接较多、乃至少量链接大量请求的场景，会出现轻微的性能损耗。
因此我们提供了可选配置`UseNetpollMode`，使用方可以根据自身的场景自行决定使用何种协程模型。