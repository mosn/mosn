package plugin

import (
	"sync"
)

var pluginFactories = make(map[string]*Client)
var pluginLock sync.Mutex

// Register called by plugin client and start up the plugin main process.
func Register(name string, config *Config) (*Client, error) {
	pluginLock.Lock()
	defer pluginLock.Unlock()

	if c, ok := pluginFactories[name]; ok {
		return c, nil
	}
	c, err := newClient(name, config)
	if err != nil {
		return nil, err
	}
	pluginFactories[name] = c

	return c, nil
}
