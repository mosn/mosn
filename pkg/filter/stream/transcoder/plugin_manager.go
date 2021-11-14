package transcoder

import (
	"errors"
	"sync"

	"mosn.io/mosn/pkg/log"
)

var transferPluginInstance *TransferPluginManager

func init() {
	transferPluginInstance = NewTransferPluginManager()
}

type TransferPluginManager struct {
	plugins sync.Map
}

func NewTransferPluginManager() *TransferPluginManager {
	return &TransferPluginManager{}
}

func GetInstanceTransferPluginManger() *TransferPluginManager {
	return transferPluginInstance
}

func (trm *TransferPluginManager) AddOrUpdateTransferPlugin(listenerName string, config []*TranscoderGoPlugin) error {
	if config == nil {
		return errors.New("plugin is empty")
	}
	if rule, ok := trm.plugins.LoadOrStore(listenerName, config); ok {
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("update transfer plugin,old:%v,new:%s", rule, config)
		}
	}
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("add transfer plugin,plugin:%s", config)
	}
	return nil
}

func (trm *TransferPluginManager) DeleteTransferPlugin(listenerName string) {
	trm.plugins.Delete(listenerName)
}

func (trm *TransferPluginManager) GetTransferPlugin(listenerName string) ([]*TranscoderGoPlugin, bool) {
	config, ok := trm.plugins.Load(listenerName)
	if ok {
		return config.([]*TranscoderGoPlugin), true
	}
	return nil, false
}
