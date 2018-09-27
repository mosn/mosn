// Copyright 2016 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by a BSD-style license.

// Package gxkafka encapsulates some kafka functions based on github.com/Shopify/sarama.
// MOD : 2016-06-01 18:00
package gxkafka

import (
	"log"
	// "fmt"
)

import (
	// "github.com/Shopify/sarama"
	"github.com/Shopify/sarama"
	sc "github.com/bsm/sarama-cluster"
	"github.com/wvanbergen/kazoo-go"
)

type (
	// Consumer will invoke @ProduceMessageCallback when got message
	// @msg: consumer message
	// @preOffset: @msg's previous message's offset in the same partition.
	//             If @msg is this partition's first message, its preOffset is 0.
	ConsumerMessageCallback func(msg *sarama.ConsumerMessage, preOffset int64)
	// Consumer will invoke @ConsumerErrorCallback when got error
	ConsumerErrorCallback func(error)
	// Consumer will invoke @ConsumerNotification when got notification
	ConsumerNotificationCallback func(*sc.Notification)
	// AsyncProducer will invoke @ProduceMessageCallback when got sucess message response.
	ProducerMessageCallback func(*sarama.ProducerMessage)
	// AsyncProducer will invoke @ProduceErrorCallback when got error message response
	ProducerErrorCallback func(*sarama.ProducerError)
)

func GetBrokerList(zkHosts string) ([]string, error) {
	var (
		config  = kazoo.NewConfig()
		zkNodes []string
	)

	// fmt.Println("zkHosts:", zkHosts)
	zkNodes, config.Chroot = kazoo.ParseConnectionString(zkHosts)
	kz, err := kazoo.NewKazoo(zkNodes, config)
	if err != nil {
		log.Fatal("[ERROR] Failed to connect to the zookeeper cluster:", err)
		return nil, err
	}
	defer kz.Close()

	brokerList, err := kz.BrokerList()
	// fmt.Printf("broker list:%#v\n", brokerList)
	if err != nil {
		log.Fatal("[ERROR] Failed to retrieve Kafka broker list from zookeeper{%s}, error{%s}", zkNodes, err)
		return nil, err
	}

	return brokerList, nil
}
