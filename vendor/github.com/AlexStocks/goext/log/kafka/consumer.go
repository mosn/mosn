// Copyright 2016 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by a BSD-style license.

// Package gxkafka encapsulates some kafka functions based on github.com/Shopify/sarama.
// MOD : 2016-06-01 05:57
package gxkafka

import (
	"fmt"
	"sync"
)

import (
	"github.com/AlexStocks/goext/sync"
	Log "github.com/AlexStocks/log4go"
	"github.com/Shopify/sarama"
	sc "github.com/bsm/sarama-cluster"
)

const (
	OFFSETS_PROCESSING_TIMEOUT_SECONDS = 10e9
	OFFSETS_COMMIT_INTERVAL            = 10e9
)

// MessageCallback is a short notation of a callback function for incoming Kafka message.
type (
	Consumer interface {
		Start() error
		Commit(*sarama.ConsumerMessage)
		Stop()
	}

	consumer struct {
		consumerGroup string
		brokers       []string
		topics        []string
		clientID      string
		msgCb         ConsumerMessageCallback
		errCb         ConsumerErrorCallback
		ntfCb         ConsumerNotificationCallback

		// cg   *consumergroup.ConsumerGroup
		cg   *sc.Consumer
		done chan gxsync.Empty
		wg   sync.WaitGroup
	}
)

func dftConsumerErrorCallback(err error) {
	Log.Error("kafka consumer error:%+v", err)
}

func dtfConsumerNotificationCallback(note *sc.Notification) {
	Log.Info("kafka consumer Rebalanced: %+v", note)
}

// NewConsumer constructs a consumer.
// @clientID should applied for sarama.validID [sarama config.go:var validID = regexp.MustCompile(`\A[A-Za-z0-9._-]+\z`)]
// the following explanation is deprecated.(2017-03-07)
// NewConsumer 之所以不能直接以brokers当做参数，是因为/wvanderbergen/kafka/consumer用到了consumer group，
// 各个消费者的信息要存到zk中
func NewConsumer(
	clientID string,
	brokers []string,
	topicList []string,
	consumerGroup string,
	msgCb ConsumerMessageCallback,
	errCb ConsumerErrorCallback,
	ntfCb ConsumerNotificationCallback,
) (Consumer, error) {

	if consumerGroup == "" || len(brokers) == 0 || len(topicList) == 0 || msgCb == nil {
		return nil, fmt.Errorf("@consumerGroup:%s, @brokers:%s, @topicList:%s, msgCb:%v",
			consumerGroup, brokers, topicList, msgCb)
	}

	if errCb == nil {
		errCb = dftConsumerErrorCallback
	}

	if ntfCb == nil {
		ntfCb = dtfConsumerNotificationCallback
	}

	return &consumer{
		consumerGroup: consumerGroup,
		brokers:       brokers,
		topics:        topicList,
		clientID:      clientID,
		msgCb:         msgCb,
		errCb:         errCb,
		ntfCb:         ntfCb,
		done:          make(chan gxsync.Empty),
	}, nil
}

// Start runs the process of consuming. It is blocking.
func (c *consumer) Start() error {
	var (
		err    error
		config *sc.Config
	)

	config = sc.NewConfig()
	config.ClientID = c.clientID
	config.Group.Return.Notifications = true
	// receive consumer error. just log it.
	config.Consumer.Return.Errors = true
	// 这个属性仅仅影响第一次启动时候获取value的offset，
	// 后面再次启动的时候如果其他config不变则会根据上次commit的offset开始获取value
	// config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.MaxProcessingTime = OFFSETS_PROCESSING_TIMEOUT_SECONDS
	// config.Consumer.Offsets.CommitInterval = OFFSETS_COMMIT_INTERVAL
	// The retention duration for committed offsets. If zero, disabled
	// (in which case the `offsets.retention.minutes` option on the
	// broker will be used).
	config.Consumer.Offsets.Retention = 0
	if c.cg, err = sc.NewConsumer(c.brokers, c.consumerGroup, c.topics, config); err != nil {
		return err
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run()
		Log.Info("consumer message processing goroutine done!")
	}()

	return nil
}

func (c *consumer) run() {
	var (
		err     error
		offsets map[string]map[int32]int64
		note    *sc.Notification
	)

	offsets = make(map[string]map[int32]int64)
	for {
		select {
		case message := <-c.cg.Messages():
			if offsets[message.Topic] == nil {
				offsets[message.Topic] = make(map[int32]int64)
			}

			if offsets[message.Topic][message.Partition] != 0 &&
				offsets[message.Topic][message.Partition] != message.Offset-1 {

				Log.Warn("Unexpected offset on %s:%d. Expected %d, found %d, diff %d.\n",
					message.Topic, message.Partition, offsets[message.Topic][message.Partition]+1,
					message.Offset, message.Offset-offsets[message.Topic][message.Partition]+1)
			}

			// message -> PushNotification
			c.msgCb(message, offsets[message.Topic][message.Partition])
			offsets[message.Topic][message.Partition] = message.Offset

		// Errors returns a read channel of errors that occur during offset management, if
		// enabled. By default, errors are logged and not returned over this channel.
		case err = <-c.cg.Errors():
			c.errCb(err)

		case note = <-c.cg.Notifications():
			c.ntfCb(note)

		case <-c.done:
			if err := c.cg.Close(); err != nil {
				Log.Warn("Error closing the consumer:%v", err)
			}

			return
		}
	}
}

func (c *consumer) Commit(message *sarama.ConsumerMessage) {
	c.cg.MarkOffset(message, "")
	// 	Log.Debug("Consuming message {%v-%v-%v}, commit over!",
	// 		message.Topic, gxstrings.String(message.Key), gxstrings.String(message.Value))
}

func (c *consumer) Stop() {
	close(c.done)
	c.wg.Wait()
}
