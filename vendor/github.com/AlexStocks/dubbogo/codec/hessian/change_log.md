## 说明 ##
---
> 目前gohessian支持hessian protocol 2.0。

## develop list ##
---
* 本项目其实是从2016/08/21开始进行改造测试的工作了，但2016/10/28才上传到github上，所以dev list只能从29号开始记了*

### 2018-05-07 ###
---
- 1 fmt.Errorf -> juju/errors.Errorf

### 2018-04-27 ###
---
- 1 improvement: pkg/errors -> juju/errors
- 2 bug fix:  ReflectResponse 中返回值为struct的时候对返回值（reflect.Value）转换为struct 对象
- 3 improvement: rewrite decBinary

### 2018-04-26 ###
---
- 1 bug fix: encInt64在v值介于 [xd8-xef]   时，应该是 “加上” BC_LONG_ZERO；


### 2017-10-16 ###
---
- 1 添加解析dubbo response相关代码；

### 2017-10-15 ###
---
- 1 添加dubbo request相关代码；

### 2017-10-12 ###
---
- 1 添加enum decode代码；
- 2 添加ref解析；

### 2017-06-13 ###
---
- 1 fix掉decList的bug，做完单测。

### 2017-06-05 ###
---
- 1 重写hessian2 enc/dec后，补充了单测

### 2016-11-04 ###
---
- 1 添加hessian2支持

### 2016-10-28 ###
---
- 1 添加github.com/AlexStocks/gohessian/encode.go:encMapByReflect函数，以编码诸如map[int]string等非map[Any]Any类型的map

### 2016-10-29 ###
---
- 1 修改 github.com/AlexStocks/gohessian/encode.go:encMap & github.com/AlexStocks/gohessian/encode.go:encMapByReflect 两个函数，当map为空的时候防止在buf里面形成垃圾数据

