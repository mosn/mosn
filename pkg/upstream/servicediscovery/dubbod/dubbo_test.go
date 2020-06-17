package dubbod

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/go-chi/chi"
	registry "github.com/mosn/registry/dubbo"
	dubbocommon "github.com/mosn/registry/dubbo/common"
	dubboconsts "github.com/mosn/registry/dubbo/common/constant"
	"github.com/mosn/registry/dubbo/remoting"
	zkreg "github.com/mosn/registry/dubbo/zookeeper"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/upstream/cluster"
	_ "mosn.io/mosn/pkg/upstream/cluster"
)

func init() {
	Init()

	monkey.Patch(zkreg.NewZkRegistry, func(url *dubbocommon.URL) (registry.Registry, error) {
		return &registry.BaseRegistry{}, nil
	})

	monkey.PatchInstanceMethod(reflect.TypeOf(&registry.BaseRegistry{}), "Register", func(r *registry.BaseRegistry, conf *dubbocommon.URL) error {
		return nil
	})

	monkey.PatchInstanceMethod(reflect.TypeOf(&registry.BaseRegistry{}), "Subscribe", func(r *registry.BaseRegistry, url *dubbocommon.URL, notifyListener registry.NotifyListener) error {
		// do nothing
		return nil
	})

	cluster.NewClusterManagerSingleton(nil, nil)

}

func TestNotify(t *testing.T) {
	// notify provider addition
	// the cluster should have 1 host
	var l = listener{}
	var event = registry.ServiceEvent{
		Action: remoting.EventTypeAdd,
		Service: *dubbocommon.NewURLWithOptions(
			dubbocommon.WithParams(url.Values{
				dubboconsts.INTERFACE_KEY: []string{"com.mosn.test.UserProvider"},
			},
			),
		)}

	l.Notify(&event)

	event = registry.ServiceEvent{
		Action: remoting.EventTypeDel,
		Service: *dubbocommon.NewURLWithOptions(
			dubbocommon.WithParams(url.Values{
				dubboconsts.INTERFACE_KEY: []string{"com.mosn.test.UserProvider"},
			},
			),
		)}

	l.Notify(&event)

	event = registry.ServiceEvent{
		Action: remoting.EventTypeUpdate,
		Service: *dubbocommon.NewURLWithOptions(
			dubbocommon.WithParams(url.Values{
				dubboconsts.INTERFACE_KEY: []string{"com.mosn.test.UserProvider"},
			},
			),
		)}

	l.Notify(&event)
}

func TestPubBindFail(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/pub", publish)

	req, _ := http.NewRequest("POST", "/pub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, fail)
}

func TestPub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/pub", publish)

	req, _ := http.NewRequest("POST", "/pub", strings.NewReader(`
	{
		"registry": {
			"addr": "127.0.0.1:2181",
			"type": "zookeeper"
		},
		"service": {
			"group": "",
			"interface": "com.ikurento.user",
			"methods": [
				"GetUser",
				"GetProfile",
				"kkk"
			],
			"name": "UserProvider",
			"port": "20000",
			"version": ""
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, succ)
}

func TestSubBindFail(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/sub", subscribe)

	req, _ := http.NewRequest("POST", "/sub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, fail)
}

func TestSub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/sub", subscribe)

	req, _ := http.NewRequest("POST", "/sub", strings.NewReader(`
	{
		"registry": {
			"addr": "127.0.0.1:2181",
			"type": "zookeeper"
		},
		"service": {
			"group": "",
			"interface": "com.ikurento.user",
			"methods": [
				"GetUser"
			],
			"name": "UserProvider"
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)

	var res resp
	err := json.Unmarshal(w.Body.Bytes(), &res)
	assert.Nil(t, err)
	assert.Equal(t, res.Errno, succ)
}

func TestUnsub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/unsub", unsubscribe)

	req, _ := http.NewRequest("POST", "/unsub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)
}

func TestUnpub(t *testing.T) {
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/unpub", unpublish)

	req, _ := http.NewRequest("POST", "/unpub", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, w.Code, 200)
}
