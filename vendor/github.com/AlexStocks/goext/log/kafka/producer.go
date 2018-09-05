// Copyright 2016 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by a BSD-style license.

// Package gxkafka encapsulates some kafka functions based on github.com/Shopify/sarama.
// MOD: 2016-06-01 05:57
package gxkafka

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/AlexStocks/goext/sync"
	"github.com/AlexStocks/goext/time"
	Log "github.com/AlexStocks/log4go"
	"github.com/Shopify/sarama"
)

const (
	HASH = iota + 1
	RANDOM
)

//////////////////////////////////////////////////////////
// Sync Producer
//////////////////////////////////////////////////////////

// Producer is interface for sending messages to Kafka.
type Producer interface {
	SendMessage(topic string, key interface{}, message interface{}) (int32, int64, error)
	SendBytes(topic string, key []byte, message []byte) (int32, int64, error)
	Stop()
}

type producer struct {
	// 此处不存储topic，不能把producer和某个topic绑定，否则就不能给其他topic发送消息。
	// topic    string
	producer sarama.SyncProducer
}

// NewProducer constructs a new SyncProducer for give brokers addresses.
// @clientID should applied for sarama.validID [sarama config.go:var validID = regexp.MustCompile(`\A[A-Za-z0-9._-]+\z`)]
// @updateMetaDataInterval is in second. to keep socket connection alive. its value shoule be less than connections.max.idle.ms.
// @compressionType pls note that the version of kafka should >= V0_10_0_0 if you wanna use CompressionLZ4.
func NewProducer(
	clientID string,
	brokers []string,
	partitionMethod int,
	waitForAllAck bool,
	updateMetaDataInterval int,
	compressionType sarama.CompressionCodec,
) (Producer, error) {

	if clientID == "" || brokers == nil || len(brokers) == 0 ||
		compressionType < sarama.CompressionNone || sarama.CompressionLZ4 < compressionType {
		return &producer{}, fmt.Errorf("@clientID:%s, @brokers:%s", clientID, brokers)
	}

	var partitionerConstructor sarama.PartitionerConstructor
	switch partitionMethod {
	case HASH:
		partitionerConstructor = sarama.NewHashPartitioner
	case RANDOM:
		partitionerConstructor = sarama.NewRandomPartitioner
	default:
		return &producer{}, fmt.Errorf("Partition method %d not supported.", partitionMethod)
	}

	var kafkaConfig = sarama.NewConfig()
	kafkaConfig.ClientID = clientID
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Partitioner = partitionerConstructor
	if waitForAllAck {
		kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	} else {
		kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	}
	kafkaConfig.Producer.Compression = compressionType

	kafkaConfig.Metadata.RefreshFrequency = gxtime.TimeSecondDuration(float64(updateMetaDataInterval))

	kafkaProducer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	return &producer{producer: kafkaProducer}, nil
}

func (p *producer) SendMessage(topic string, key interface{}, message interface{}) (partition int32, offset int64, err error) {
	msg, err := json.Marshal(message)
	if err != nil {
		return -1, -1, fmt.Errorf("cannot marshal message %v: %v", message, err)
	}

	var keyEncoder, valueEncoder sarama.Encoder
	valueEncoder = sarama.ByteEncoder(msg)
	var producerMessage = sarama.ProducerMessage{
		Topic: topic,
		Value: valueEncoder,
	}
	if key != nil {
		keyByte, err := json.Marshal(key)
		if err != nil {
			return -1, -1, fmt.Errorf("cannot marshal key%v: %v", key, err)
		}

		keyEncoder = sarama.ByteEncoder(keyByte)
		// keyEncoder = sarama.StringEncoder(key)
		producerMessage.Key = keyEncoder
	}
	partition, offset, err = p.producer.SendMessage(&producerMessage)
	if err != nil {
		return -1, -1, fmt.Errorf("cannot send message %v: %v", message, err)
	}

	return partition, offset, nil
}

func (p *producer) SendBytes(topic string, key []byte, message []byte) (partition int32, offset int64, err error) {
	var keyEncoder, valueEncoder sarama.Encoder
	valueEncoder = sarama.ByteEncoder(message)
	var producerMessage = sarama.ProducerMessage{
		Topic: topic,
		Value: valueEncoder,
	}
	if key != nil {
		keyEncoder = sarama.ByteEncoder(key)
		producerMessage.Key = keyEncoder
	}
	partition, offset, err = p.producer.SendMessage(&producerMessage)
	if err != nil {
		return -1, -1, fmt.Errorf("cannot send message %v: %v", message, err)
	}

	return partition, offset, nil
}

func (p *producer) Stop() {
	p.producer.Close()
}

//////////////////////////////////////////////////////////
// Async Producer
//////////////////////////////////////////////////////////

// Producer is interface for sending messages to Kafka.
type AsyncProducer interface {
	SendMessage(topic string, key interface{}, message interface{}, metadata interface{}) error
	SendBytes(topic string, key []byte, message []byte, metadata interface{})
	Start()
	Stop()
	Terminate()
}

type asyncProducer struct {
	producer sarama.AsyncProducer
	sucMsgCb ProducerMessageCallback
	errMsgCb ProducerErrorCallback

	msgNum  int64
	sucNum  int64
	failNum int64

	done chan gxsync.Empty
	wg   sync.WaitGroup
}

// NewAsyncProducer constructs a new AsyncProducer for give brokers addresses.
// @clientID should applied for sarama.validID [sarama config.go:var validID = regexp.MustCompile(`\A[A-Za-z0-9._-]+\z`)]
// @updateMetaDataInterval is in second. to keep socket connection alive. its value shoule be less than connections.max.idle.ms.
// @compressionType pls note that the version of kafka should >= V0_10_0_0 if you wanna use CompressionLZ4.
func NewAsyncProducer(
	clientID string,
	brokers []string,
	partitionMethod int,
	waitForAllAck bool,
	updateMetaDataInterval int,
	compressionType sarama.CompressionCodec,
	successfulMessageCallback ProducerMessageCallback,
	errorCallback ProducerErrorCallback,
) (AsyncProducer, error) {

	if clientID == "" || brokers == nil || len(brokers) == 0 ||
		compressionType < sarama.CompressionNone || sarama.CompressionLZ4 < compressionType {
		return &asyncProducer{}, fmt.Errorf("@clientID:%s, @brokers:%s, @compressionType:%d", clientID, brokers, compressionType)
	}

	var partitionerConstructor sarama.PartitionerConstructor
	switch partitionMethod {
	case HASH:
		partitionerConstructor = sarama.NewHashPartitioner
	case RANDOM:
		partitionerConstructor = sarama.NewRandomPartitioner
	default:
		return &asyncProducer{}, fmt.Errorf("Partition method %d not supported.", partitionMethod)
	}

	var kafkaConfig = sarama.NewConfig()
	kafkaConfig.ClientID = clientID
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Partitioner = partitionerConstructor
	kafkaConfig.Producer.Compression = compressionType
	if waitForAllAck {
		kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	} else {
		kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	}

	kafkaConfig.Metadata.RefreshFrequency = gxtime.TimeSecondDuration(float64(updateMetaDataInterval))

	kafkaProducer, err := sarama.NewAsyncProducer(brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	return &asyncProducer{
		producer: kafkaProducer,
		sucMsgCb: successfulMessageCallback,
		errMsgCb: errorCallback,
		done:     make(chan gxsync.Empty),
	}, nil
}

func (p *asyncProducer) SendMessage(
	topic string,
	key interface{},
	message interface{},
	metadata interface{},
) (err error) {

	msg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("cannot marshal message %v: %v", message, err)
	}

	var (
		valueEncoder    sarama.Encoder
		producerMessage *sarama.ProducerMessage
	)
	valueEncoder = sarama.ByteEncoder(msg)
	producerMessage = &sarama.ProducerMessage{
		Topic:    topic,
		Value:    valueEncoder,
		Metadata: metadata,
	}
	if key != nil {
		keyByte, err := json.Marshal(key)
		if err != nil {
			return fmt.Errorf("cannot marshal key%v: %v", key, err)
		}

		producerMessage.Key = sarama.ByteEncoder(keyByte)
	}

	p.producer.Input() <- producerMessage
	atomic.AddInt64(&(p.msgNum), 1)

	return nil
}

func (p *asyncProducer) SendBytes(
	topic string,
	key []byte,
	message []byte,
	metadata interface{}) {

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic:    topic,
		Key:      sarama.ByteEncoder(key),
		Value:    sarama.ByteEncoder(message),
		Metadata: metadata,
	}
	atomic.AddInt64(&(p.msgNum), 1)
}

func (p *asyncProducer) Start() {
	p.wg.Add(1)
	go func() {
		var (
			errMsg *sarama.ProducerError
			msg    *sarama.ProducerMessage
		)

		defer p.wg.Done()

	LOOP:
		for {
			select {
			case errMsg = <-p.producer.Errors():
				p.errMsgCb(errMsg)
				atomic.AddInt64(&(p.failNum), 1)
			case msg = <-p.producer.Successes():
				p.sucMsgCb(msg)
				atomic.AddInt64(&(p.sucNum), 1)
			case <-p.done:
				break LOOP
			}
		}
	}()
}

func (p *asyncProducer) Stop() {
	var (
		all, suc, fail int64
		waitTime       time.Duration
	)

	waitTime = time.Duration(1e8)
LOOP:
	for {
		all = atomic.LoadInt64(&(p.msgNum))
		suc = atomic.LoadInt64(&(p.sucNum))
		fail = atomic.LoadInt64(&(p.failNum))
		// fmt.Printf("all:%d, suc:%d, fail:%d\n", all, suc, fail)
		// if len(p.producer.Errors()) == 0 && len(p.producer.Successes()) == 0 {
		if all == (suc + fail) {
			Log.Info("all:%d, suc:%d, fail:%d\n", all, suc, fail)
			close(p.done)
			break LOOP
		}
		time.Sleep(waitTime)
		waitTime += time.Duration(1e8)
		if time.Duration(2e9) < waitTime {
			waitTime = time.Duration(2e9)
		}
	}

	p.wg.Wait()

	// Close will invoke AsyncClose
	if err := p.producer.Close(); err != nil {
		Log.Error("async producer Close() error:%v", err)
	}
}

func (p *asyncProducer) Terminate() {
	close(p.done)

	p.wg.Wait()

	// Close will invoke AsyncClose
	if err := p.producer.Close(); err != nil {
		Log.Error("async producer Close() error:%v", err)
	}
}
