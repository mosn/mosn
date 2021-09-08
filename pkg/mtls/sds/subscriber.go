package sds

import (
	"sync"
	"time"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type SdsSubscriber struct {
	provider             types.SecretProvider
	reqQueue             chan string
	sdsConfig            interface{}
	sdsStreamClient      SdsStreamClient
	sdsStreamClientMutex sync.RWMutex
	stopChannel          chan struct{}
}

type SdsStreamClient interface {
	// Send creates a secret discovery request with name, and send it to server
	Send(name string) error
	// Recv receive a secret discovert response, handle it and send a ack response
	Recv(provider types.SecretProvider, callback func()) error
	// Stop stops a stream client
	Stop()
}

func NewSdsSubscriber(provider types.SecretProvider, sdsConfig interface{}) *SdsSubscriber {
	return &SdsSubscriber{
		provider:    provider,
		sdsConfig:   sdsConfig,
		reqQueue:    make(chan string, 10),
		stopChannel: make(chan struct{}),
	}
}

var SubscriberRetryPeriod = 3 * time.Second

func (subscribe *SdsSubscriber) Start() {
	subscribe.connect()

	utils.GoWithRecover(func() {
		subscribe.sendRequestLoop()
	}, nil)
	utils.GoWithRecover(func() {
		subscribe.receiveResponseLoop()
	}, nil)

}

func (subscribe *SdsSubscriber) getSdsStreamClient(config interface{}) error {
	subscribe.sdsStreamClientMutex.Lock()
	defer subscribe.sdsStreamClientMutex.Unlock()
	if subscribe.sdsStreamClient != nil {
		return nil
	}
	streamClient, err := GetSdsStreamClient(subscribe.sdsConfig)
	if err != nil {
		return err
	}
	subscribe.sdsStreamClient = streamClient
	return nil
}

func (subscribe *SdsSubscriber) connect() {
	for {
		if err := subscribe.getSdsStreamClient(subscribe.sdsConfig); err != nil {
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		log.DefaultLogger.Infof("[sds][subscribe] init sds stream client success")
		return
	}
}

func (subscribe *SdsSubscriber) Stop() {
	close(subscribe.stopChannel)
}

func (subscribe *SdsSubscriber) SendSdsRequest(name string) {
	subscribe.reqQueue <- name
}

var retryInterval = time.Second

func (subscribe *SdsSubscriber) sendRequestLoop() {
	for {
		select {
		case <-subscribe.stopChannel:
			log.DefaultLogger.Errorf("[xds] [sds subscriber] send request loop closed")
			subscribe.cleanSdsStreamClient()
			return
		case name := <-subscribe.reqQueue:
			for {
				subscribe.sdsStreamClientMutex.RLock()
				clt := subscribe.sdsStreamClient
				subscribe.sdsStreamClientMutex.RUnlock()

				if clt == nil {
					log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed, send request failed")
					time.Sleep(retryInterval)
					continue
				}

				if err := clt.Send(name); err != nil {
					log.DefaultLogger.Alertf("sds.subscribe.request", "[xds] [sds subscriber] send sds request fail , resource name = %v", name)
					time.Sleep(retryInterval)
					continue
				}
				break
			}
		}
	}
}

func (subscribe *SdsSubscriber) receiveResponseLoop() {
	for {
		select {
		case <-subscribe.stopChannel:
			log.DefaultLogger.Errorf("[xds] [sds subscriber]  receive response loop closed")
			return
		default:
			subscribe.sdsStreamClientMutex.RLock()
			clt := subscribe.sdsStreamClient
			subscribe.sdsStreamClientMutex.RUnlock()

			if clt == nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed, reconnect after 1s")
				time.Sleep(retryInterval)
				subscribe.reconnect()
				continue
			}

			if err := clt.Recv(subscribe.provider, sdsPostCallback); err != nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] get resp error: %v,reconnect  1s", err)
				time.Sleep(retryInterval)
				subscribe.reconnect()
				continue
			}

		}
	}
}
func (subscribe *SdsSubscriber) reconnect() {
	subscribe.cleanSdsStreamClient()
	subscribe.connect()
}

func (subscribe *SdsSubscriber) cleanSdsStreamClient() {
	subscribe.sdsStreamClientMutex.Lock()
	defer subscribe.sdsStreamClientMutex.Unlock()
	if subscribe.sdsStreamClient != nil {
		subscribe.sdsStreamClient.Stop()
		subscribe.sdsStreamClient = nil
	}
	log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed")
}
