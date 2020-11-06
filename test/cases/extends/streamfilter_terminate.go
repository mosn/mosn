package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream("mock_terminate", CreateTerminateFilter)
}

var once sync.Once
var globalStore *mockStore

func CreateTerminateFilter(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	once.Do(func() {
		globalStore = &mockStore{}
		v, _ := conf["address"]
		addr, ok := v.(string)
		if !ok {
			addr = "127.0.0.1:12345"
		}
		globalStore.Start(addr)
	})
	return &TerminateFilterFactory{}, nil
}

type TerminateFilterFactory struct {
}

func (f *TerminateFilterFactory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewTerminateFilter(ctx)
	callbacks.AddStreamReceiverFilter(filter, api.AfterChooseHost) // if mosn cannot choose a host, mosn do response directly.
}

// TerminateFilter mocks use terminate to stop proxyed request.
type TerminateFilter struct {
	handler api.StreamReceiverFilterHandler
}

func NewTerminateFilter(ctx context.Context) *TerminateFilter {
	return &TerminateFilter{}
}

func (f *TerminateFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	globalStore.Store(handler)
}

func (f *TerminateFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return api.StreamFilterContinue // returns FilterContinue to make sure request proxyed
}

func (f *TerminateFilter) OnDestroy() {
}

// mock termiante requests
type mockStore struct {
	mutex sync.Mutex
	store []api.StreamReceiverFilterHandler
}

func (s *mockStore) Store(handler api.StreamReceiverFilterHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.store = append(s.store, handler)
}

func (s *mockStore) run(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	success := 0
	failed := 0
	for _, h := range s.store {
		if h.TerminateStream(504) {
			success++
		} else {
			failed++
		}
	}
	s.store = s.store[:0] // clean
	result := fmt.Sprintf("terminate requests, success %d, failed %d", success, failed)
	w.Write([]byte(result))
}

func (s *mockStore) Start(addr string) {
	http.HandleFunc("/", s.run)
	go http.ListenAndServe(addr, nil)
}
