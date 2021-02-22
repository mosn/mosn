// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	globalCfg   = NewDefaultConfig()
	initLogOnce sync.Once
)

func ResetGlobalConfig(config *Entity) {
	globalCfg = config
}

// InitConfigWithYaml loads general configuration from the YAML file under provided path.
func InitConfigWithYaml(filePath string) (err error) {
	// Initialize general config and logging module.
	if err = applyYamlConfigFile(filePath); err != nil {
		return err
	}
	return OverrideConfigFromEnvAndInitLog()
}

// applyYamlConfigFile loads general configuration from the given YAML file.
func applyYamlConfigFile(configPath string) error {
	// Priority: system environment > YAML file > default config
	if util.IsBlank(configPath) {
		// If the config file path is absent, Sentinel will try to resolve it from the system env.
		configPath = os.Getenv(ConfFilePathEnvKey)
	}
	if util.IsBlank(configPath) {
		configPath = DefaultConfigFilename
	}
	// First Sentinel will try to load config from the given file.
	// If the path is empty (not set), Sentinel will use the default config.
	return loadGlobalConfigFromYamlFile(configPath)
}

func OverrideConfigFromEnvAndInitLog() error {
	// Then Sentinel will try to get fundamental config items from system environment.
	// If present, the value in system env will override the value in config file.
	err := overrideItemsFromSystemEnv()
	if err != nil {
		return err
	}

	defer logging.Info("[Config] Print effective global config", "globalConfig", *globalCfg)
	// Configured Logger is the highest priority
	if configLogger := Logger(); configLogger != nil {
		err = logging.ResetGlobalLogger(configLogger)
		if err != nil {
			return err
		}
		return nil
	}

	logDir := LogBaseDir()
	if len(logDir) == 0 {
		logDir = GetDefaultLogDir()
	}
	if err := initializeLogConfig(logDir, LogUsePid()); err != nil {
		return err
	}
	logging.Info("[Config] App name resolved", "appName", AppName())
	return nil
}

func loadGlobalConfigFromYamlFile(filePath string) error {
	if filePath == DefaultConfigFilename {
		if _, err := os.Stat(DefaultConfigFilename); err != nil {
			//use default globalCfg.
			return nil
		}
	}
	_, err := os.Stat(filePath)
	if err != nil && !os.IsExist(err) {
		return err
	}
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(content, globalCfg)
	if err != nil {
		return err
	}
	logging.Info("[Config] Resolving Sentinel config from file", "file", filePath)
	return checkConfValid(&(globalCfg.Sentinel))
}

func overrideItemsFromSystemEnv() error {
	if appName := os.Getenv(AppNameEnvKey); !util.IsBlank(appName) {
		globalCfg.Sentinel.App.Name = appName
	}

	if appTypeStr := os.Getenv(AppTypeEnvKey); !util.IsBlank(appTypeStr) {
		appType, err := strconv.ParseInt(appTypeStr, 10, 32)
		if err != nil {
			return err
		} else {
			globalCfg.Sentinel.App.Type = int32(appType)
		}
	}

	if addPidStr := os.Getenv(LogNamePidEnvKey); !util.IsBlank(addPidStr) {
		addPid, err := strconv.ParseBool(addPidStr)
		if err != nil {
			return err
		} else {
			globalCfg.Sentinel.Log.UsePid = addPid
		}
	}

	if logDir := os.Getenv(LogDirEnvKey); !util.IsBlank(logDir) {
		globalCfg.Sentinel.Log.Dir = logDir
	}
	return checkConfValid(&(globalCfg.Sentinel))
}

func initializeLogConfig(logDir string, usePid bool) (err error) {
	if logDir == "" {
		return errors.New("invalid empty log path")
	}

	initLogOnce.Do(func() {
		if err = util.CreateDirIfNotExists(logDir); err != nil {
			return
		}
		err = reconfigureRecordLogger(logDir, usePid)
	})
	return err
}

func reconfigureRecordLogger(logBaseDir string, withPid bool) error {
	filePath := filepath.Join(logBaseDir, logging.RecordLogFileName)
	if withPid {
		filePath = filePath + ".pid" + strconv.Itoa(os.Getpid())
	}

	fileLogger, err := logging.NewSimpleFileLogger(filePath)
	if err != nil {
		return err
	}
	// Note: not thread-safe!
	if err := logging.ResetGlobalLogger(fileLogger); err != nil {
		return err
	}

	logging.Info("[Config] Log base directory", "baseDir", logBaseDir)

	return nil
}

func GetDefaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, logging.DefaultDirName)
}

func AppName() string {
	return globalCfg.AppName()
}

func AppType() int32 {
	return globalCfg.AppType()
}

func Logger() logging.Logger {
	return globalCfg.Logger()
}

func LogBaseDir() string {
	return globalCfg.LogBaseDir()
}

// LogUsePid returns whether the log file name contains the PID suffix.
func LogUsePid() bool {
	return globalCfg.LogUsePid()
}

func MetricLogFlushIntervalSec() uint32 {
	return globalCfg.MetricLogFlushIntervalSec()
}

func MetricLogSingleFileMaxSize() uint64 {
	return globalCfg.MetricLogSingleFileMaxSize()
}

func MetricLogMaxFileAmount() uint32 {
	return globalCfg.MetricLogMaxFileAmount()
}

func SystemStatCollectIntervalMs() uint32 {
	return globalCfg.SystemStatCollectIntervalMs()
}

func UseCacheTime() bool {
	return globalCfg.UseCacheTime()
}

func GlobalStatisticIntervalMsTotal() uint32 {
	return globalCfg.GlobalStatisticIntervalMsTotal()
}

func GlobalStatisticSampleCountTotal() uint32 {
	return globalCfg.GlobalStatisticSampleCountTotal()
}

func GlobalStatisticBucketLengthInMs() uint32 {
	return globalCfg.GlobalStatisticIntervalMsTotal() / GlobalStatisticSampleCountTotal()
}

func MetricStatisticIntervalMs() uint32 {
	return globalCfg.MetricStatisticIntervalMs()
}
func MetricStatisticSampleCount() uint32 {
	return globalCfg.MetricStatisticSampleCount()
}
