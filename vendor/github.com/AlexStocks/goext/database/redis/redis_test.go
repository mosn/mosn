// ref: https://github.com/alexstocks/go-sentinel/blob/master/sentinel_test.go
package gxredis

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/garyburd/redigo/redis"
)

const (
	HOST_IP = "192.168.11.100"
)

func TestSentinel_GetInstances(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380"},
	)
	defer st.Close()

	instances, err := st.GetInstances()
	if err != nil {
		t.Errorf("st.GetInstances, error:%#v\n", err)
		t.FailNow()
	}

	for idx, inst := range instances {
		inst_str, _ := json.Marshal(inst)
		t.Logf("idx:%d, instance:%s\n", idx, inst_str)
		err = st.Discover(inst.Name, []string{"127.0.0.1"})
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
	}

	addrs := st.GetSentinels()
	t.Logf("sentinel instances:%#v\n", addrs)
}

func TestSentinel_GetInstanceNames(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380"},
	)
	defer st.Close()

	names, err := st.GetInstanceNames()
	if err != nil {
		t.Errorf("st.GetInstanceNames, error:%#v\n", err)
		t.FailNow()
	}
	t.Logf("sentinel instance names:%#v\n", names)
}
func TestSentinel_AddInstance(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380", HOST_IP + ":26381", HOST_IP + ":26382"},
	)
	defer st.Close()

	//to find all sentinel addresses
	instances, err := st.GetInstances()
	if err != nil {
		t.Errorf("st.GetInstances, error:%#v\n", err)
		t.FailNow()
	}

	for _, inst := range instances {
		// 如果所有的sentinel都在一个机器上部署着，如果不加上excludeIPArray参数，
		// 则执行完结果是 [192.168.10.100:26380 192.168.10.100:26381 192.168.10.100:26382 127.0.0.1:26382 127.0.0.1:26381]
		err = st.Discover(inst.Name, []string{"127.0.0.1"})
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
	}
	t.Log(st.Addrs)

	st.RemoveInstance("meta")
	inst := RawInstance{
		Name:            "meta",
		Addr:            &IPAddr{IP: HOST_IP, Port: 6000},
		Epoch:           2,
		Sdowntime:       10,
		FailoverTimeout: 450,
	}
	err = st.AddInstance(inst)
	if err != nil {
		t.Errorf("RemoveInstance(meta) = error:%#v", err)
	}
}

func TestSentinel_RemoveInstance(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380", HOST_IP + ":26381", HOST_IP + ":26382"},
	)
	defer st.Close()

	inst := RawInstance{
		Name:            "meta",
		Addr:            &IPAddr{IP: HOST_IP, Port: 6000},
		Epoch:           2,
		Sdowntime:       10,
		FailoverTimeout: 450,
	}
	st.AddInstance(inst)
	err := st.RemoveInstance("meta")
	if err != nil {
		t.Errorf("RemoveInstance(meta) = error:%#v", err)
	}
}

func TestSentinel_GetConn(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380"},
	)
	defer st.Close()

	instances, err := st.GetInstances()
	if err != nil {
		t.Errorf("st.GetInstances, error:%#v\n", err)
		t.FailNow()
	}

	for i, inst := range instances {
		err = st.Discover(inst.Name, []string{"127.0.0.1"})
		if err != nil {
			t.Log(err)
			t.FailNow()
		}

		conn, _ := st.GetConnByRole(fmt.Sprintf("%s:%d", inst.Master.IP, inst.Master.Port), RR_Master)
		if conn == nil {
			fmt.Println("get conn fail, ", inst.Master.IP, inst.Master.Port)
			continue
		}
		defer conn.Close()
		s, err := redis.String(conn.Do("INFO"))
		if err != nil {
			fmt.Println("do command error:", err)
			fmt.Printf("do command error for master addr{idx:%s, addr:%#v}", i, inst.Master)
			continue
		}
		fmt.Printf("idx:%s, addr:%#v, info:%#v", i, inst.Master, s)
		time.Sleep(1 * time.Second)
	}
}

func TestSentinel_MakeMasterSwitchSentinelWatcher(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380"},
	)
	defer st.Close()

	instances, err := st.GetInstances()
	if err != nil {
		t.Errorf("st.GetInstances, error:%#v\n", err)
		t.FailNow()
	}

	for _, inst := range instances {
		err = st.Discover(inst.Name, []string{"127.0.0.1"})
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
	}

	wg := &sync.WaitGroup{}
	watcher, err := st.MakeMasterSwitchSentinelWatcher()
	w, err := watcher.Watch()
	_ = w
	go func() {
		defer wg.Done()
		wg.Add(1)
		for addr := range w {
			t.Logf("redis instance switch: %#v\n", addr)
		}
		fmt.Println("watch exit")
	}()
	time.Sleep(30 * time.Second)
	fmt.Println("close")
	watcher.Close()
	wg.Wait()
}

func TestSentinel_MakeSdownSentinelWatcher(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380"},
	)
	defer st.Close()

	instances, err := st.GetInstances()
	if err != nil {
		t.Errorf("st.GetInstances, error:%#v\n", err)
		t.FailNow()
	}

	for _, inst := range instances {
		err = st.Discover(inst.Name, []string{"127.0.0.1"})
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
	}

	wg := &sync.WaitGroup{}
	watcher, err := st.MakeSdownSentinelWatcher()
	w, err := watcher.Watch()
	_ = w
	go func() {
		defer wg.Done()
		wg.Add(1)
		for info := range w {
			fmt.Printf("watch info:%s\n", info)
			t.Logf("redis slave down: %#v\n", info)
		}
		fmt.Println("watch exit")
	}()
	time.Sleep(30 * time.Second)
	fmt.Println("close")
	watcher.Close()
	wg.Wait()
}

func TestSentinel_Transaction(t *testing.T) {
	st := NewSentinel(
		[]string{HOST_IP + ":26380"},
	)
	defer st.Close()

	instances, err := st.GetInstances()
	if err != nil {
		t.Errorf("st.GetInstances, error:%#v\n", err)
		t.FailNow()
	}

	for _, inst := range instances {
		err = st.Discover(inst.Name, []string{"127.0.0.1"})
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
	}

	conn, _ := st.GetConnByRole(net.JoinHostPort(HOST_IP, "6000"), RR_Master)
	if conn == nil {
		t.Errorf("get host %s conn fail", net.JoinHostPort(HOST_IP, "6000"))
		t.FailNow()
	}

	defer func() {
		if err != nil {
			conn.Do("discard")
		}
		conn.Close()
	}()

	key := "testk"
	value := "testv"
	// tx进行过程中，key发生任何改变（如原来不存在，tx过程中被创建；或者原来存在，tx过程被删除或者值被修改），tx就会失败
	if _, err = conn.Do("watch", key); err != nil {
		t.Errorf("watch %s, got error:%#v", key, err)
		t.FailNow()
	}

	conn.Send("multi")
	time.Sleep(10e9)
	conn.Do("Set", "testKey", value)
	conn.Do("Set", "testKey2", value)

	queued, err := conn.Do("exec")
	if err != nil {
		t.Errorf("exec error:%#v", err)
		t.FailNow()
	}
	if queued == nil {
		t.Logf("tx failed, q:%#v", queued)
	}
}
