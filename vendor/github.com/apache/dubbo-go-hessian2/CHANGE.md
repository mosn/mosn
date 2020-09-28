# Release Notes

## v1.7.0

### New Features
- add GetStackTrace method into Throwabler and its implements. [#207](https://github.com/apache/dubbo-go-hessian2/pull/207)
- catch user defined exceptions. [#208](https://github.com/apache/dubbo-go-hessian2/pull/208)
- support java8 time object. [#212](https://github.com/apache/dubbo-go-hessian2/pull/212), [#221](https://github.com/apache/dubbo-go-hessian2/pull/221)
- support test golang encoding data in java. [#213](https://github.com/apache/dubbo-go-hessian2/pull/213)
- support java.sql.Time & java.sql.Date. [#219](https://github.com/apache/dubbo-go-hessian2/pull/219)

### Enhancement
- Export function EncNull. [#225](https://github.com/apache/dubbo-go-hessian2/pull/225)

### Bugfixes
- fix eunm encode error in request. [#203](https://github.com/apache/dubbo-go-hessian2/pull/203)
- fix []byte field decoding issue. [#216](https://github.com/apache/dubbo-go-hessian2/pull/216)
- fix decoding error for map in map. [#229](https://github.com/apache/dubbo-go-hessian2/pull/229)
- fix fields name mismatch in Duration class. [#234](https://github.com/apache/dubbo-go-hessian2/pull/234)

## v1.6.0

### New Features
- ignore non-exist fields when decoding. [#201](https://github.com/apache/dubbo-go-hessian2/pull/201)

### Enhancement
- add cache in reflection to improve performance. [#179](https://github.com/apache/dubbo-go-hessian2/pull/179)
- string decode performance improvement. [#188](https://github.com/apache/dubbo-go-hessian2/pull/188)

### Bugfixes
- fix attachment lost for nil value. [#191](https://github.com/apache/dubbo-go-hessian2/pull/191)
- fix float32 accuracy issue. [#196](https://github.com/apache/dubbo-go-hessian2/pull/196)

## v1.5.0

### New Features
- support java collection.  [#161](https://github.com/apache/dubbo-go-hessian2/pull/161)

### Bugfixes
- fix skipping fields bug. [#167](https://github.com/apache/dubbo-go-hessian2/pull/167)


## v1.4.0

### New Features
- support BigInteger.  [#141](https://github.com/apache/dubbo-go-hessian2/pull/141)
- support embedded struct. [#150](https://github.com/apache/dubbo-go-hessian2/pull/150)
- flat anonymous struct field. [#154](https://github.com/apache/dubbo-go-hessian2/pull/154)

### Enhancement
- update bytes pool. [#147](https://github.com/apache/dubbo-go-hessian2/pull/147)

### Bugfixes
- fix check service.Group and service.Interface. [#138](https://github.com/apache/dubbo-go-hessian2/pull/138)
- fix can't duplicately decode Serializer object. [#144](https://github.com/apache/dubbo-go-hessian2/pull/144)
- fix bug for encTypeInt32. [#148](https://github.com/apache/dubbo-go-hessian2/pull/148)


## v1.3.0

### New Features
- support skip unregistered pojo. [#128](https://github.com/apache/dubbo-go-hessian2/pull/128)

### Enhancement
- change nil string pointer value. [#121](https://github.com/apache/dubbo-go-hessian2/pull/121)
- convert attachments to map[string]string]. [#127](https://github.com/apache/dubbo-go-hessian2/pull/127)

### Bugfixes
- fix nil bool error. [#114](https://github.com/apache/dubbo-go-hessian2/pull/114)
- fix tag parse error in decType. [#120](https://github.com/apache/dubbo-go-hessian2/pull/120)
- fix dubbo version. [#130](https://github.com/apache/dubbo-go-hessian2/pull/130)
- fix emoji encode error. [#131](https://github.com/apache/dubbo-go-hessian2/pull/131)

