# Release Notes

## v1.10.2

### Bugfixes
- fix list value not unpacked. [#300](https://github.com/apache/dubbo-go-hessian2/pull/300)

## v1.10.1

### Bugfixes
- support java integer null. [#296](https://github.com/apache/dubbo-go-hessian2/pull/296)
- fix parse basic type bug. [#298](https://github.com/apache/dubbo-go-hessian2/pull/298)
 
## v1.10.0

### New Features
- support java function param type. [#295](https://github.com/apache/dubbo-go-hessian2/pull/295)

## v1.9.5

### New Features
- support serialize UUID to string. [#285](https://github.com/apache/dubbo-go-hessian2/pull/285)
- support encode non-pointer instance for pointer POJO definition. [#289](https://github.com/apache/dubbo-go-hessian2/pull/289)

### Bugfixes
- fix POJO registration. [#287](https://github.com/apache/dubbo-go-hessian2/pull/287)
- fix EOF error check. [#288](https://github.com/apache/dubbo-go-hessian2/pull/288)
- fix go type name for list. [#290](https://github.com/apache/dubbo-go-hessian2/pull/290)

## v1.9.4

### New Features
- support wrapper classes for Java basic types. [#278](https://github.com/apache/dubbo-go-hessian2/pull/278)

### Bugfixes
- fix registration ignored for struct with same name in diff package. [#279](https://github.com/apache/dubbo-go-hessian2/pull/279)
- fix cannot encode pointer of raw type. [#283](https://github.com/apache/dubbo-go-hessian2/pull/283)

## v1.9.3

### New Features
- add new api `Encoder.ReuseBufferClean()`. [#271](https://github.com/apache/dubbo-go-hessian2/pull/271)

### Bugfixes
- fix not unpack ref holder for list. [#269](https://github.com/apache/dubbo-go-hessian2/pull/269)
- fix encode null for empty map, add map tag instead. [#275](https://github.com/apache/dubbo-go-hessian2/pull/275)
- Fix getArgType reflection value logic. [#276](https://github.com/apache/dubbo-go-hessian2/pull/276)

## v1.9.2

### New Features
- support java.util.Locale. [#264](https://github.com/apache/dubbo-go-hessian2/pull/264)

## v1.9.1

### Bugfixes
- fix repeatedly adding list type in type map. [#263](https://github.com/apache/dubbo-go-hessian2/pull/263)

## v1.9.0

### New Features
- support java UUID object. [#256](https://github.com/apache/dubbo-go-hessian2/pull/256)

### Bugfixes
- fix map decode error. [#261](https://github.com/apache/dubbo-go-hessian2/pull/261)

## v1.8.2

### Bugfixes
- fix insufficient bytes for string encoding buffers. [#255](https://github.com/apache/dubbo-go-hessian2/pull/255)

## v1.8.1

### Bugfixes
- fix get wrong javaclassname for POJO struct. [#247](https://github.com/apache/dubbo-go-hessian2/pull/247)
- fix not enough buf error when decode date. [#249](https://github.com/apache/dubbo-go-hessian2/pull/249)
- fix emoji decoding error. [#254](https://github.com/apache/dubbo-go-hessian2/pull/254)

## v1.8.0

### New Features
- support clean encoder/decoder, discard decode buffer. [#242](https://github.com/apache/dubbo-go-hessian2/pull/242)
- support encode no pojo object. [#243](https://github.com/apache/dubbo-go-hessian2/pull/243)

### Enhancement
- change value reference to ptr to improve performance. [#244](https://github.com/apache/dubbo-go-hessian2/pull/244)

### Bugfixes
- fix issue that cannot decode java generic type. [#239](https://github.com/apache/dubbo-go-hessian2/pull/239)

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
- fix enum encode error in request. [#203](https://github.com/apache/dubbo-go-hessian2/pull/203)
- fix []byte field decoding issue. [#216](https://github.com/apache/dubbo-go-hessian2/pull/216)
- fix decoding error for map in map. [#229](https://github.com/apache/dubbo-go-hessian2/pull/229)

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

