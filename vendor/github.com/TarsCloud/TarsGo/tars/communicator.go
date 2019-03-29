package tars

import (
	s "github.com/TarsCloud/TarsGo/tars/model"
	"sync"
)

// ProxyPrx interface
type ProxyPrx interface {
	SetServant(s.Servant)
}

// Dail returns servant proxy
func Dail(servant string) *ServantProxy {
	c := new(Communicator)
	c.init()
	return c.s.GetServantProxy(servant)
}

// NewCommunicator returns a new communicator. A Communicator is used for communicating with the server side which should only init once and be global!!!
func NewCommunicator() *Communicator {
	c := new(Communicator)
	c.init()
	return c
}

// Communicator struct
type Communicator struct {
	s          *ServantProxyFactory
	Client     *clientConfig
	properties sync.Map
}

func (c *Communicator) init() {
	if GetClientConfig() != nil {
		c.SetProperty("locator", GetClientConfig().Locator)
		//TODO
		c.Client = GetClientConfig()
	} else {
		c.Client = &clientConfig{
			"",
			"",
			"",
			"",
			refreshEndpointInterval,
			reportInterval,
			AsyncInvokeTimeout,
		}
	}
	c.SetProperty("netthread", 2)
	c.SetProperty("isclient", true)
	c.SetProperty("enableset", false)
	if GetServerConfig() != nil {
		c.SetProperty("netthread", GetServerConfig().netThread)
		c.SetProperty("notify", GetServerConfig().notify)
		c.SetProperty("node", GetServerConfig().Node)
		c.SetProperty("server", GetServerConfig().Server)
		c.SetProperty("isclient", false)
		if GetServerConfig().Enableset {
			c.SetProperty("enableset", true)
			c.SetProperty("setdivision", GetServerConfig().Setdivision)
		}
	}

	c.s = new(ServantProxyFactory)
	c.s.Init(c)
}

// GetLocator returns locator as string
func (c *Communicator) GetLocator() string {
	v, _ := c.GetProperty("locator")
	return v
}

// SetLocator sets locator with obj
func (c *Communicator) SetLocator(obj string) {
	c.SetProperty("locator", obj)
}

// StringToProxy sets the servant of ProxyPrx p with a string servant
func (c *Communicator) StringToProxy(servant string, p ProxyPrx) {
	p.SetServant(c.s.GetServantProxy(servant))
}

// SetProperty sets communicator property with a string key and an interface value.
// var comm *tars.Communicator
// comm = tars.NewCommunicator()
// e.g. comm.SetProperty("locator", "tars.tarsregistry.QueryObj@tcp -h ... -p ...")
func (c *Communicator) SetProperty(key string, value interface{}) {
	c.properties.Store(key, value)
}

// GetProperty returns communicator property value as string and true for key, or empty string and false for not exists key
func (c *Communicator) GetProperty(key string) (string, bool) {
	if v, ok := c.properties.Load(key); ok {
		return v.(string), ok
	}
	return "", false
}

// GetPropertyInt returns communicator property value as int and true for key, or 0 and false for not exists key
func (c *Communicator) GetPropertyInt(key string) (int, bool) {
	if v, ok := c.properties.Load(key); ok {
		return v.(int), ok
	}
	return 0, false
}

// GetPropertyBool returns communicator property value as bool and true for key, or false and false for not exists key
func (c *Communicator) GetPropertyBool(key string) (bool, bool) {
	if v, ok := c.properties.Load(key); ok {
		return v.(bool), ok
	}
	return false, false
}
