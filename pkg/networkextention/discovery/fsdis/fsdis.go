/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fsdsdiscovery

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/networkextention/discovery"
	"mosn.io/pkg/utils"
)

func init() {
	utils.GoWithRecover(func() {
		getAppsInfoFromFileSystemFile(FileSystemDiscoveryFile)
	}, nil)

	discovery.RegisterDiscovery("fsDiscovery", NewFileSystemDiscovery())
	if l, err := discovery.NewMonitorDiscovery("fsDiscovery"); err == nil {
		l.Start()
	} else {
		log.DefaultLogger.Fatalf("NewMonitorDiscovery failed: %v", err)
	}
}

const (
	// Some information about services
	// TODO support env or configurable
	FileSystemDiscoveryFile = "/home/admin/mosn/discovery/service.json"
)

const (
	GetFileSystemDiscoveryRetryLoopInterval = 1 * time.Second
)

var (
	once                               sync.Once
	fileSystemDiscoveryManagerInstance *FileSystemDiscovery
)

type appsInfo struct {
	Apps []struct {
		AppName    string `json:"app_name"`
		ServerList []struct {
			Ip   string `json:"ip"`
			Port uint32 `json:"port"`
			Site string `json:"site"`
		} `json:"server_list"`
	} `json:"apps"`
}

func getAppsInfoFromFileSystemFile(path string) {

	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.DefaultLogger.Errorf("[FileSystemDiscovery][getAppsInfoFromFileSystemFile] create watch %s failed: %v", path, err)
	}

	defer watch.Close()
	for {
		err = watch.Add(path)
		if err != nil {
			log.DefaultLogger.Errorf("[FileSystemDiscovery][getAppsInfoFromFileSystemFile] watch %s failed: %v", path, err)
			time.Sleep(GetFileSystemDiscoveryRetryLoopInterval)
		} else {
			break
		}

	}

	updateHandler(path)

	for {
		select {
		case ev := <-watch.Events:
			updateHandler(path)
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[FileSystemDiscovery][getAppsInfoFromFileSystemFile]ev.Op: %d", ev.Op)
			}
		case err := <-watch.Errors:
			{
				log.DefaultLogger.Errorf("[FileSystemDiscovery][getAppsInfoFromFileSystemFile] failed: %v", err)
				time.Sleep(GetFileSystemDiscoveryRetryLoopInterval)
				continue
			}
		}
	}

}

func updateHandler(path string) bool {
	serviceAddrInfo := make([]ServiceAddrInfo, 0, 10)
	d := GetSignalFileSystemDiscovery()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.DefaultLogger.Errorf("[FileSystemDiscovery][updateHandler] load config failed, error: %v", err)
		return false
	}

	cfg := &appsInfo{}

	err = json.Unmarshal(content, cfg)
	if err != nil {
		log.DefaultLogger.Errorf("[FileSystemDiscovery][updateHandler] json unmarshal config failed, error: %v", err)
		return false
	}

	for _, app := range cfg.Apps {
		serviceAddrInfo = serviceAddrInfo[:0]
		for _, addr := range app.ServerList {
			serviceAddrInfo = append(serviceAddrInfo, NewServiceAddrInfo(addr.Ip, addr.Port, addr.Site))
		}
		d.AddorUpdateAppServiceMonitor(app.AppName, serviceAddrInfo)
	}

	return true
}

type ServiceAddrInfo struct {
	ip   string
	site string
	port uint32
}

func NewServiceAddrInfo(ip string, port uint32, site string) ServiceAddrInfo {
	return ServiceAddrInfo{
		ip:   ip,
		port: port,
		site: site,
	}
}

func (addr ServiceAddrInfo) GetServiceIP() string {
	return addr.ip
}

func (addr ServiceAddrInfo) GetServiceSite() string {
	return addr.site
}

func (addr ServiceAddrInfo) GetServicePort() uint32 {
	return addr.port
}

type AppService struct {
	name      string
	addrsInfo []ServiceAddrInfo
	changed   uint32
}

func NewAppService(appName string, addrsInfo []ServiceAddrInfo) *AppService {
	ai := make([]ServiceAddrInfo, len(addrsInfo))
	copy(ai, addrsInfo)
	return &AppService{
		name:      appName,
		addrsInfo: ai,
		changed:   1,
	}
}

func (as *AppService) updateAddrs(addrsInfo []ServiceAddrInfo) {
	as.addrsInfo = make([]ServiceAddrInfo, len(addrsInfo))
	copy(as.addrsInfo, addrsInfo)
}

type FileSystemDiscovery struct {
	services sync.Map // store AppService pointer
	rw       sync.RWMutex
}

func GetSignalFileSystemDiscovery() *FileSystemDiscovery {
	return NewFileSystemDiscovery()
}

func NewFileSystemDiscovery() *FileSystemDiscovery {
	once.Do(func() {
		fileSystemDiscoveryManagerInstance = &FileSystemDiscovery{

			services: sync.Map{},
		}
	})
	return fileSystemDiscoveryManagerInstance
}

func (l *FileSystemDiscovery) AddorUpdateAppServiceMonitor(appName string, addrsInfo []ServiceAddrInfo) error {
	l.rw.Lock()
	defer l.rw.Unlock()
	if v, ok := l.services.Load(appName); ok {
		var appService *AppService
		appService, ok = v.(*AppService)
		if !ok {
			log.DefaultLogger.Warnf("[FileSystemDiscovery][AddorUpdateAppServiceMonitor] type assert failed AppName %s", appName)
			return errors.New("AppService type assert failed")
		}

		appService.updateAddrs(addrsInfo)
		atomic.StoreUint32(&appService.changed, 1)

	} else {
		l.services.LoadOrStore(appName, NewAppService(appName, addrsInfo))
	}
	return nil
}

func (l *FileSystemDiscovery) GetServiceAddrInfo(appName string) []discovery.ServiceAddrInfo {
	var appService *AppService
	var serviceAddrInfo []discovery.ServiceAddrInfo

	v, ok := l.services.Load(appName)
	if !ok {
		log.DefaultLogger.Warnf("[FileSystemDiscovery][GetServiceAddrInfo] not found AppName %s from services", appName)
		return serviceAddrInfo
	}

	if appService, ok = v.(*AppService); !ok {
		log.DefaultLogger.Warnf("[FileSystemDiscovery][GetServiceAddrInfo] type assert failed AppName %s", appName)
		return serviceAddrInfo
	}

	if len(appService.addrsInfo) == 0 {
		log.DefaultLogger.Warnf("[FileSystemDiscovery][GetServiceAddrInfo] get empty server list from AppName %s", appName)
		return serviceAddrInfo
	}

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[FileSystemDiscovery][GetServiceAddrInfo] antvipClient.GetRealServers(domain:%s) = realServers:%+v",
			appName, serviceAddrInfo)
	}

	l.rw.RLock()
	defer l.rw.RUnlock()

	serviceAddrInfo = make([]discovery.ServiceAddrInfo, 0, len(appService.addrsInfo))

	for index, _ := range appService.addrsInfo {
		serviceAddrInfo = append(serviceAddrInfo, appService.addrsInfo[index])
	}

	return serviceAddrInfo
}

func (l *FileSystemDiscovery) GetAllServiceName() []string {
	serviceList := make([]string, 0, 1)

	l.services.Range(func(key, value interface{}) bool {
		serviceList = append(serviceList, key.(string))
		return true
	})

	return serviceList
}

func (l *FileSystemDiscovery) CheckAndResetServiceChange(appName string) bool {
	var appService *AppService

	if v, ok := l.services.Load(appName); ok {
		if appService, ok = v.(*AppService); ok {
			return atomic.CompareAndSwapUint32(&appService.changed, 1, 0)
		}
	}

	log.DefaultLogger.Errorf("[FileSystemDiscovery][CheckAndResetServiceChange] not found appService: %s", appName)

	return false
}

func (l *FileSystemDiscovery) SetServiceChangeRetry(appName string) bool {
	var appService *AppService
	if v, ok := l.services.Load(appName); ok {
		if appService, ok = v.(*AppService); ok {
			atomic.StoreUint32(&appService.changed, 1)
			return true
		}
	}

	log.DefaultLogger.Errorf("[FileSystemDiscovery][SetServiceChangeRetry] not found appService: %s", appName)

	return false
}
