# goext
---
*my golang sdk package*

## dev list
---

- 2018/08/16
  > Improvement
  * Add ServiceAttr as Watch filter

- 2018/08/13
  > Improvement
  * use ServiceArray

- 2018/08/09
  > feature
  * add gxset from scylla/go-set

- 2018/08/07
  > feature
  * add gxrsync from c4milo/gsync

- 2018/07/17
  > feature
  * add consistent hash from lafikl/consistent

- 2018/07/07
  > feature
  * add sync/atmoic from github.com/uber-go/atomic
  * add container/btree from github.com/google/btree
  * add io/kv/disk

- 2018/05/07
  > feature
  * add container/hashtable

- 2018/05/02
  > bug fix
  * use value copy in for-loop

- 2018/04/28
  > bug fix
  * add ServiceAttr GeneralEqual to filter service

- 2018/04/21
  > feature
  * add reuseport from github.com/kavu/go_reuseport

- 2018/04/20
  > feature
  * add timespan from github.com/senseyeio/spaniel/

- 2018/04/19
  > feature
  * add handle etcd restart
  * add cache selector

- 2018/04/18
  > improvement
  * test gxetcd.Client

- 2018/04/16
  > feature
  * add ServiceAttr Marshal/Unmarshal

- 2018/04/11
  > feature
  * add gxetcd

- 2018/04/09
  > feature
  * add gxstrings/Builder

- 2018/04/04
  > bugfix
  * check error when unzip/ungzip

  > feature
  * add WriteFile, ReadTxt -> ReadFile

- 2018/03/29
  > feature
  * add strings/IsNil

- 2018/03/29
  > feature
  * add compress/zip
  * add compress/gzip
  * use juju/errors as default errors

- 2018/03/27
  > feature
  * add json logger

- 2018/03/26
  > feature
  * add async logger

- 2018/03/22
  > feature
  * add gxjson

- 2018/03/21
  > feature
  * add gxnet.IsSameAddr

- 2018/03/14
  > feature
  * add sync/drwlock
  * add time/gxtime parser
  * add sync/trylock

- 2018/01/26
  > feature
  * add context

- 2018/01/25
  > feature
  * add gxtimeticker

- 2018/01/24
  > feature
  * add gxtime sleep

- 2018/01/21
  > feature
  * add gxtime time wheel

- 2017/12/02
  > feature
  * add c like api strftime in time

- 2017/11/26
  > improvement
  * add GenSinaShortURLByGoogd because the source field is invalid in SinaShortURL

- 2017/11/05
  > feature
  * add runtime/pprof by gops

- 2017/10/31
  > feature
  * add short url api

- 2017/10/30
  > improvement
  * add leak check on file header: // +build !leak

- 2017/10/28
  > feature
  * gxlog.PrettyStruct -> gxlog.PrettyString in log/elasticsearch.go

- 2017/10/24
  > feature
  * add runtime/goroutine_pool

- 2017/10/12
  > feature
  * add gxlog.ColorSprint & gxlog.ColorSprintln & ColorSprintf

- 2017/10/11
  > feature
  * add os.GetPkgPath

- 2017/09/21
  > bugfix
  * check redis master is available or not in databases/redis/GetInstances

- 2017/09/19
  > feature
  * subscribe +sdown redis channel to get crashed redis instance

- 2017/08/21
  > improvement
  * use gogoprotobuf for redis

- 2017/08/21
  > feature
  * add deque

- 2017/08/11
  > feature
  * add redis

- 2017/08/05
  > feature
  * add unbouned channel

- 2017/07/20
  > feature
  * colorful print

- 2017/07/09
  > feature
  * add HashMap

- 2017/06/12
  > feature
  * add updateMetaDataInterval for kafka producer

- 2017/06/10
  > feature
  * add compression for kafka producer

- 2017/05/02
  > feature
  * add gxtime.Wheel:Now()

- 2017/04/25
  > feature
  * add elasticsearch bulk insert

- 2017/04/24

   > feature
   * add pretty struct in log/pretty.go

   > improvement
   * add mysql prepared statement for gxdriver

- 2017/04/20

   > feature
   * add Time2Unix & Unix2Time & UnixString2Time in time

- 2017/04/13

    > feature
    * add container/gxset

- 2017/04/02
    * feature: add log/elasticsearch
    * improvement: add gxstrings:CheckByteArray

- 2017/03/07
    * improvement: change ConsumerMessageCallback parameter list and delete its return value.
    * improvement: add ConsumerErrorCallback and ConsumerNotificationCallback for kafka consumer
    * improvement: delete NewConsuemrWithZk and NewProducerWithZk

- 2017/03/06
    * improvement: change Log repo to my log4go

- 2017/03/01
    * bug fix: use github.com/bsm/sarama-cluster to construct consumer group on kafka v0.10 instead of github.com/wvanbergen/kafka
    * bug fix: use right license file

- 2017/02/07
    * add broadcaster sync/broadcast.go
    * do not use this again and modify all

- 2017/01/22
    * add asynchronous kafka producer in log/kafka/producer.go

- 2017/01/20
    * modify log/kafka/consumer.go

- 2017/01/14
    * add log/kafka to encapsulate kafka producer/consumer functions
    * add math/rand/red_packet.go to provide tencent red packet algorithm

- 2017/01/12
    * move github.com/AlexStocks/dubbogo/common/net.go to github.com/AlexStocks/goext/net/ip.go
    * move github.com/AlexStocks/dubbogo/common/misc.go(Contains, ArrayRemoveAt, RandStringBytesMaskImprSrc) to github.com/AlexStocks/goext/strings/strings.go
    * move github.com/AlexStocks/dubbogo/common/misc.go(Future) to github.com/AlexStocks/goext/time/time.go
    * move github.com/AlexStocks/dubbogo/common/misc.go(Goid) to github.com/AlexStocks/goext/runtime/mprof.go(GoID)

- 2016/10/23
    * move github.com/AlexStocks/pool to github.com/AlexStocks/goext/sync/pool

- 2016/10/14
    * add goext/time/YMD
    * add goext/time/PrintTime

- 2016/10/01
    * add goext/bitmap
    * optimize goext/time timer by bitmap

- 2016/09/27
    * add uuid in strings
    * add ReadTxt & String & Slice in io/ioutil

- 2016/09/26
    * add CountWatch in goext/time
    * add HostAddress in goext/net
    * add RandString & RanddigitString & UUID for goext/math/rand
    * add Wheel for goext/time

- 2016/09/22
    * add multiple loggers test for goext/log

- 2016/09/21
    * os
    * log

- 2016/08/30
    * sync/trylock
    * sync/semaphore
    * time/time

- 2015/05/03
    * container/xorlist
