package config

const (
	// UnknownProjectName represents the "default" value
	// that indicates the project name is absent.
	UnknownProjectName = "unknown_go_service"

	ConfFilePathEnvKey = "SENTINEL_CONFIG_FILE_PATH"
	AppNameEnvKey      = "SENTINEL_APP_NAME"
	AppTypeEnvKey      = "SENTINEL_APP_TYPE"
	LogDirEnvKey       = "SENTINEL_LOG_DIR"
	LogNamePidEnvKey   = "SENTINEL_LOG_USE_PID"

	DefaultConfigFilename       = "sentinel.yml"
	DefaultAppType        int32 = 0

	DefaultMetricLogFlushIntervalSec   uint32 = 1
	DefaultMetricLogSingleFileMaxSize  uint64 = 1024 * 1024 * 50
	DefaultMetricLogMaxFileAmount      uint32 = 8
	DefaultSystemStatCollectIntervalMs uint32 = 1000
)
