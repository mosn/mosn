# Release Notes

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

