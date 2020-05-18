package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

// InitConfig loads general configuration from the given file.
func InitConfig(configPath string) error {
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
	err := LoadFromYamlFile(configPath)
	if err != nil {
		return err
	}
	// Then Sentinel will try to get fundamental config items from system environment.
	// If present, the value in system env will override the value in config file.
	err = overrideItemsFromSystemEnv()
	if err != nil {
		return err
	}
	err = InitializeLogConfig(LogBaseDir(), LogUsePid())
	if err != nil {
		return err
	}

	logging.GetDefaultLogger().Infof("App name resolved: %s", AppName())

	return nil
}

func LoadFromYamlFile(filePath string) error {
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
	logging.GetDefaultLogger().Infof("Resolving Sentinel config from file: %s", filePath)
	return checkValid(&(globalCfg.Sentinel))
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
	return checkValid(&(globalCfg.Sentinel))
}

func InitializeLogConfig(logDir string, usePid bool) (err error) {
	if logDir == "" {
		return errors.New("Invalid empty log path")
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
	logDir := util.AddPathSeparatorIfAbsent(logBaseDir)
	filePath := logDir + logging.RecordLogFileName
	if withPid {
		filePath = filePath + ".pid" + strconv.Itoa(os.Getpid())
	}

	defaultLogger := logging.GetDefaultLogger()
	if defaultLogger == nil {
		return errors.New("Unexpected state: defaultLogger == nil")
	}
	logFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}

	// Note: not thread-safe!
	logging.ResetDefaultLogger(log.New(logFile, "", log.LstdFlags), logging.DefaultNamespace)
	fmt.Println("INFO: log base directory is: " + logDir)

	return nil
}

func GetDefaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return util.AddPathSeparatorIfAbsent(home) + logging.DefaultDirName
}

func AppName() string {
	return globalCfg.Sentinel.App.Name
}

func AppType() int32 {
	return globalCfg.Sentinel.App.Type
}

func LogBaseDir() string {
	return globalCfg.Sentinel.Log.Dir
}

// LogUsePid returns whether the log file name contains the PID suffix.
func LogUsePid() bool {
	return globalCfg.Sentinel.Log.UsePid
}

func MetricLogFlushIntervalSec() uint32 {
	return globalCfg.Sentinel.Log.Metric.FlushIntervalSec
}

func MetricLogSingleFileMaxSize() uint64 {
	return globalCfg.Sentinel.Log.Metric.SingleFileMaxSize
}

func MetricLogMaxFileAmount() uint32 {
	return globalCfg.Sentinel.Log.Metric.MaxFileCount
}

func SystemStatCollectIntervalMs() uint32 {
	return globalCfg.Sentinel.Stat.System.CollectIntervalMs
}
