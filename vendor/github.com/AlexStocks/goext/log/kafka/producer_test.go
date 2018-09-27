package gxkafka

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	"github.com/Shopify/sarama"
)

func init() {
	sarama.Logger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)
}

type Message struct {
	EventName string `json:"event_name,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	IP        string `json:"ip,omitempty"`
	PlanID    int64  `json:"plan_id,omitempty"`
	UUID      string `json:"uuid,omitempty"`
	OS        int    `json:"os,omitempty"`
	AppID     int    `json:"app_id,omitempty"`
	Token     string `json:"token,omitempty"`
	Status    string `json:"status,omitempty"`
}

func TestKafkaProducer(t *testing.T) {
	var (
		id        = "producer-client-id"
		topic     = "test1"
		brokers   = []string{"127.0.0.1:9092"}
		num       int
		err       error
		partition int32
		offset    int64
		producer  Producer
		message   Message
	)

	if producer, err = NewProducer(id, brokers, HASH, true, 45, sarama.CompressionGZIP); err != nil {
		t.Errorf("NewProducerWithZk(id:%s, brokers:%s) = err{%v}", id, brokers, err)
	}
	defer producer.Stop()

	message.EventName = "event"
	message.Timestamp = time.Now().Unix()
	message.IP = "127.0.0.1"
	message.PlanID = 123
	message.UUID = "a-b-c-d-e"
	message.OS = 2
	message.AppID = 1
	message.Token = "1234"
	message.Status = "good"

	num = 10
	for i := 0; i < num; i++ {
		message.UUID = "hello:" + strconv.Itoa(i)
		partition, offset, err = producer.SendMessage(topic, message.UUID, message)
		if err != nil {
			t.Fatalf("FAILED to produce message:%v, err:%v", message, err)
		}
		t.Logf("send telemetry message:%#v, kafka partition:%d, kafka offset:%d",
			message, partition, offset)
		time.Sleep(1e9)
	}
}

// go test -bench=. -run=BenchmarkProducer_SendMessage
// 2000000000	         0.11 ns/op

// go test -v -bench BenchmarkProducer_SendMessage -run=^a
// 1000000000	         0.66 ns/op
// github.com/AlexStocks/goext/log/kafka	12.042s
func BenchmarkProducer_SendMessage(b *testing.B) {
	var (
		id      = "producer-client-id"
		topic   = "test1"
		brokers = []string{"127.0.0.1:9092"}
		num     int
		err     error
		// partition int32
		// offset    int64
		producer Producer
		message  Message
	)

	b.StopTimer()
	if producer, err = NewProducer(id, brokers, HASH, true, 45, sarama.CompressionGZIP); err != nil {
		b.Errorf("NewProducerWithZk(id:%s, brokers:%s) = err{%v}", id, brokers, err)
	}
	defer producer.Stop()

	message.EventName = "event"
	message.Timestamp = time.Now().Unix()
	message.IP = "127.0.0.1"
	message.PlanID = 123
	message.UUID = "a-b-c-d-e"
	message.OS = 2
	message.AppID = 1
	message.Token = "1234"
	message.Status = "good"

	b.StartTimer()
	num = 1000
	for i := 0; i < num; i++ {
		message.UUID = "hello:" + strconv.Itoa(i)
		// partition, offset, err = producer.SendMessage(topic, message.UUID, message)
		_, _, err = producer.SendMessage(topic, message.UUID, message)
		if err != nil {
			b.Fatalf("FAILED to produce message:%v, err:%v", message, err)
		}
		// b.Logf("send telemetry message:%#v, kafka partition:%d, kafka offset:%d",
		// 	message, partition, offset)
		// time.Sleep(1e9)
	}
	b.StopTimer()
	gxlog.CDebug("finished!!!!!!!!!!!!!!!!!!!")
}

type MessageMetadata struct {
	EnqueuedAt time.Time
}

func (mm *MessageMetadata) Latency() time.Duration {
	return time.Since(mm.EnqueuedAt)
}

func TestAsyncKafkaProducer(t *testing.T) {
	var (
		id              = "producer-client-id"
		topic           = "test1"
		brokers         = []string{"127.0.0.1:9092"}
		err             error
		latency         time.Duration
		totalLatency    time.Duration
		messageNum      int
		enqueued        int
		successes       int
		failures        int
		rate            float64
		messageStart    time.Time
		messageDuration time.Duration
		msgCallback     ProducerMessageCallback
		errCallback     ProducerErrorCallback
		producer        AsyncProducer
		message         Message
	)

	msgCallback = func(message *sarama.ProducerMessage) {
		log.Printf("send msg{%v, %v} response{topic:%s, partition:%d, offset:%d}\n",
			message.Key, message.Value, message.Topic, message.Partition, message.Offset)
		totalLatency += message.Metadata.(*MessageMetadata).Latency()
		successes++
	}

	errCallback = func(err *sarama.ProducerError) {
		log.Printf("send msg:%v failed. error:%v\n", err.Msg, err.Error())
		failures++
	}

	if producer, err = NewAsyncProducer(id, brokers, HASH, true, 45, sarama.CompressionLZ4, msgCallback, errCallback); err != nil {
		t.Errorf("NewAsyncProducerWithZk(zk:%s) = err{%v}", brokers, err)
	}
	producer.Start()

	message.EventName = "event"
	message.Timestamp = time.Now().Unix()
	message.IP = "127.0.0.1"
	message.PlanID = 123
	message.UUID = "a-b-c-d-e"
	message.OS = 2
	message.AppID = 1
	message.Token = "1234"
	message.Status = "good"

	messageNum = 10
	messageStart = time.Now()
	for i := 0; i < messageNum; i++ {
		message.UUID = "hello:" + strconv.Itoa(i)
		if err = producer.SendMessage(topic, message.UUID, message, &MessageMetadata{EnqueuedAt: time.Now()}); err != nil {
			t.Fatalf("FAILED to produce message:%v, err:%v", message, err)
		}
		enqueued++
		time.Sleep(1e9)
	}

	producer.Stop()
	// 发送消息的速率
	messageDuration = time.Since(messageStart)
	rate = float64(successes) / (float64(messageDuration) / float64(time.Second))
	latency = totalLatency / time.Duration(successes)
	log.Printf("Success{Rate: %0.2f/s; latency: %0.2fms}\n",
		rate, float64(latency)/float64(time.Millisecond))
	log.Printf("Enqueued: %d; Produced: %d; Failed: %d; Send rate %0.2fm/s.\n",
		enqueued, successes, failures, float64(messageNum)/(float64(messageDuration)/float64(time.Second)))
}

// go test -v -bench BenchmarkAsyncProducer_SendMessage -run=^a
// 2000000000	         0.05 ns/op
// ok  	github.com/AlexStocks/goext/log/kafka	1.071s
func BenchmarkAsyncProducer_SendMessage(b *testing.B) {
	var (
		id           = "producer-client-id"
		zk           = "127.0.0.1:2181/kafka"
		topic        = "test1"
		err          error
		totalLatency time.Duration
		messageNum   int
		enqueued     int
		successes    int
		failures     int
		brokers      []string
		msgCallback  ProducerMessageCallback
		errCallback  ProducerErrorCallback
		producer     AsyncProducer
		message      Message
	)

	b.StopTimer()
	msgCallback = func(message *sarama.ProducerMessage) {
		// log.Printf("send msg{%v, %v} response{topic:%s, partition:%d, offset:%d}\n",
		//	message.Key, message.Value, message.Topic, message.Partition, message.Offset)
		totalLatency += message.Metadata.(*MessageMetadata).Latency()
		successes++
	}

	errCallback = func(err *sarama.ProducerError) {
		log.Printf("send msg:%v failed. error:%v\n", err.Msg, err.Error())
		failures++
	}

	if brokers, err = GetBrokerList(zk); err != nil {
		b.Fatalf("Failed to get broker list: %v", err)
	}

	if producer, err = NewAsyncProducer(id, brokers, HASH, true, 45, sarama.CompressionLZ4, msgCallback, errCallback); err != nil {
		b.Errorf("NewAsyncProducerWithZk(zk:%s) = err{%v}", zk, err)
	}
	producer.Start()

	message.EventName = "event"
	message.Timestamp = time.Now().Unix()
	message.IP = "127.0.0.1"
	message.PlanID = 123
	message.UUID = "a-b-c-d-e"
	message.OS = 2
	message.AppID = 1
	message.Token = "1234"
	message.Status = "good"

	b.StartTimer()
	messageNum = 1000
	for i := 0; i < messageNum; i++ {
		message.UUID = "hello:" + strconv.Itoa(i)
		if err = producer.SendMessage(topic, message.UUID, message, &MessageMetadata{EnqueuedAt: time.Now()}); err != nil {
			b.Fatalf("FAILED to produce message:%v, err:%v", message, err)
		}
		enqueued++
	}

	producer.Stop()
	b.StopTimer()
}
