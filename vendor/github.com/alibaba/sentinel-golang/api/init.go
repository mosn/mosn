package api

import (
	"fmt"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/log/metric"
	"github.com/alibaba/sentinel-golang/core/system"
)

// InitDefault initializes Sentinel using the configuration from system
// environment and the default value.
func InitDefault() error {
	return initSentinel("")
}

// Init loads Sentinel general configuration from the given YAML file
// and initializes Sentinel.
func Init(configPath string) error {
	return initSentinel(configPath)
}

func initSentinel(configPath string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	// Initialize general config and logging module.
	if err = config.InitConfig(configPath); err != nil {
		return err
	}
	return initCoreComponents()
}

func initCoreComponents() (err error) {
	if err = metric.InitTask(); err != nil {
		return err
	}

	system.InitCollector(config.SystemStatCollectIntervalMs())
	return err
}
