package tars

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	debugutil "github.com/TarsCloud/TarsGo/tars/util/debug"
	logger "github.com/TarsCloud/TarsGo/tars/util/rogger"
)

//Admin struct
type Admin struct {
}

//Shutdown shutdown all servant by admin
var (
	isShutdownbyadmin int32 = 0
)

func (a *Admin) Shutdown() error {
	atomic.StoreInt32(&isShutdownbyadmin, 1)
	go graceShutdown()
	return nil
}

//Notify handler for cmds from admin
func (a *Admin) Notify(command string) (string, error) {
	cmd := strings.Split(command, " ")
	go ReportNotifyInfo("AdminServant::notify:" + cmd[0])
	switch cmd[0] {
	case "tars.viewversion":
		return GetServerConfig().Version, nil
	case "tars.setloglevel":
		switch cmd[1] {
		case "INFO":
			logger.SetLevel(logger.INFO)
		case "WARN":
			logger.SetLevel(logger.WARN)
		case "ERROR":
			logger.SetLevel(logger.ERROR)
		case "DEBUG":
			logger.SetLevel(logger.DEBUG)
		case "NONE":
			logger.SetLevel(logger.OFF)
		}
		return fmt.Sprintf("%s succ", command), nil
	case "tars.dumpstack":
		debugutil.DumpStack(true, "stackinfo")
		return fmt.Sprintf("%s succ", command), nil
	case "tars.loadconfig":
		cfg := GetServerConfig()
		remoteConf := NewRConf(cfg.App, cfg.Server, cfg.BasePath)
		_, err := remoteConf.GetConfig(cmd[1])
		if err != nil {
			return fmt.Sprintf("Getconfig Error!: %s", cmd[1]), err
		}
		return fmt.Sprintf("Getconfig Success!: %s", cmd[1]), nil

	case "tars.connection":
		return fmt.Sprintf("%s not support now!", command), nil
	case "tars.gracerestart":
		graceRestart()
		return "restart gracefully!", nil
	case "tars.pprof":
		port := ":8080"
		timeout := time.Second * 600
		if len(cmd) > 1 {
			port = ":" + cmd[1]
		}
		if len(cmd) > 2 {
			t, _ := strconv.ParseInt(cmd[2], 10, 64)
			if 0 < t && t < 3600 {
				timeout = time.Second * time.Duration(t)
			}
		}
		cfg := GetServerConfig()
		addr := cfg.LocalIP + port
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
			mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
			mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
			mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
			mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
			s := &http.Server{Addr: addr, Handler: mux}
			TLOG.Info("start serve pprof ", addr)
			go s.ListenAndServe()
			time.Sleep(timeout)
			s.Shutdown(context.Background())
			TLOG.Info("stop serve pprof ", addr)
		}()
		return "see http://" + addr + "/debug/pprof/", nil
	default:
		if fn, ok := adminMethods[cmd[0]]; ok {
			return fn(command)
		}
		return fmt.Sprintf("%s not support now!", command), nil
	}
}

//RegisterAdmin register admin functions
func RegisterAdmin(name string, fn adminFn) {
	adminMethods[name] = fn
}
