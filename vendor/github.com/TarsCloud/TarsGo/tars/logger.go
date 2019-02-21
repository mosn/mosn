package tars

import (
	"path/filepath"

	"github.com/TarsCloud/TarsGo/tars/util/rogger"
)

// GetLogger Get a logger
func GetLogger(name string) *rogger.Logger {
	cfg, name := logName(name)
	lg := rogger.GetLogger(name)
	logpath := filepath.Join(cfg.LogPath, cfg.App, cfg.Server)
	lg.SetFileRoller(logpath, int(cfg.LogNum), int(cfg.LogSize))
	return lg
}

func logName(name string) (*serverConfig, string) {
	cfg := GetServerConfig()
	if name == "" {
		name = cfg.App + "." + cfg.Server
	} else {
		name = cfg.App + "." + cfg.Server + "_" + name
	}
	return cfg, name
}

// GetDayLogger Get a logger roll by day
func GetDayLogger(name string, numDay int) *rogger.Logger {
	cfg, name := logName(name)
	lg := rogger.GetLogger(name)
	logpath := filepath.Join(cfg.LogPath, cfg.App, cfg.Server)
	lg.SetDayRoller(logpath, numDay)
	return lg
}

// GetHourLogger Get a logger roll by hour
func GetHourLogger(name string, numHour int) *rogger.Logger {
	cfg, name := logName(name)
	lg := rogger.GetLogger(name)
	logpath := filepath.Join(cfg.LogPath, cfg.App, cfg.Server)
	lg.SetHourRoller(logpath, numHour)
	return lg
}

//GetRemoteLogger returns a remote logger
func GetRemoteLogger(name string) *rogger.Logger {
	cfg := GetServerConfig()
	lg := rogger.GetLogger(name)
	if !lg.IsConsoleWriter() {
		return lg
	}
	remoteWriter := NewRemoteTimeWriter()
	var set string
	if cfg.Enableset {
		set = cfg.Setdivision
	}

	remoteWriter.InitServerInfo(cfg.App, cfg.Server, name, set)
	lg.SetWriter(remoteWriter)
	return lg

}
