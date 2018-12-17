package sink

import (
	"testing"
	"sync"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"time"
	"net/http"
	"io/ioutil"
	"log"
)

type testAction int

const (
	countInc        testAction = iota
	countDec
	gaugeUpdate
	histogramUpdate
)

// test concurrently add statisic data
// should get the right data from prometheus
func TestPrometheusMetrics(t *testing.T) {
	testCases := []struct {
		typ         string
		namespace   string
		key         string
		action      testAction
		actionValue int64
	}{
		{"t1", "ns1", "k1", countInc, 1},
		{"t1", "ns1", "k1", countDec, 1},
		{"t1", "ns1", "k2", countInc, 1},
		{"t1", "ns1", "k3", gaugeUpdate, 1},
		{"t1", "ns1", "k4", histogramUpdate, 1},
		{"t1", "ns1", "k4", histogramUpdate, 2},
		{"t1", "ns1", "k4", histogramUpdate, 3},
		{"t1", "ns1", "k4", histogramUpdate, 4},
		{"t1", "ns2", "k1", countInc, 1},
		{"t1", "ns2", "k2", countInc, 2},
		{"t1", "ns2", "k3", gaugeUpdate, 3},
		{"t1", "ns2", "k4", histogramUpdate, 2},
		{"t2", "ns1", "k1", countInc, 1},
	}
	wg := sync.WaitGroup{}
	for i := range testCases {
		wg.Add(1)
		go func(i int) {
			time.Sleep(300 * time.Duration(i) * time.Millisecond)
			tc := testCases[i]
			s := stats.NewStats(tc.typ, tc.namespace)
			switch tc.action {
			case countInc:
				s.Counter(tc.key).Inc(tc.actionValue)
			case countDec:
				s.Counter(tc.key).Dec(tc.actionValue)
			case gaugeUpdate:
				s.Gauge(tc.key).Update(tc.actionValue)
			case histogramUpdate:
				s.Histogram(tc.key).Update(tc.actionValue)
			}
			wg.Done()
		}(i)
	}
	//init prom
	flushInteval := time.Millisecond * 500
	sink := NewPromeSink(&PromConfig{
		FlushInterval: flushInteval,
	})

	times := 0
	tc := http.Client{}
	stopChan := make(chan bool)

	go func() {
		for {
			select {

			case <-time.Tick(flushInteval):
				allRegistry := stats.GetAllRegistries()
				for _, reg := range allRegistry {
					sink.Flush(reg)
				}

				resp, _ := tc.Get("http://127.0.0.1:8080/metrics")
				body, _ := ioutil.ReadAll(resp.Body)
				times++
				log.Printf("========= %d times prom metrics pull =========\n", times)
				log.Print(string(body))
			case <-stopChan:
				break
			}
		}
	}()


	wg.Wait()
	stopChan <- true
	typs := stats.LisTypes()
	if !(len(typs) == 2 &&
		typs[0] == "t1" &&
		typs[1] == "t2") {
		t.Error("types record error")
	}
}
