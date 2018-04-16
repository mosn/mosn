package server

import (
	"os"
	"time"
	"sync"
	"strconv"
	"io/ioutil"
	"os/signal"
	"syscall"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"strings"
)

const (
	MosnBasePath = string(os.PathSeparator) + "home" + string(os.PathSeparator) +
		"admin" + string(os.PathSeparator) + "mosn"
)

func init() {
	writePidFile()

	catchSignals()
}

var (
	pidFile string

	onProcessExit []func()

	gracefulTimeout time.Duration

	BaseFolder string

	shutdownCallbacksOnce sync.Once

	shutdownCallbacks []func() error
)

func writePidFile() error {
	pidFile = MosnBasePath + string(os.PathSeparator) + "pid.log"
	pid := []byte(strconv.Itoa(os.Getpid()) + "\n")

	os.MkdirAll(MosnBasePath, 0644);

	return ioutil.WriteFile(pidFile, pid, 0644)
}

func catchSignals() {
	catchSignalsCrossPlatform()
	catchSignalsPosix()
}

func catchSignalsCrossPlatform() {
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP,
			syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

		for sig := range sigchan {
			log.DefaultLogger.Println(sig, " received!")
			switch sig {
			case syscall.SIGQUIT:
				// quit
				for _, f := range onProcessExit {
					f() // only perform important cleanup actions
				}
				os.Exit(0)
			case syscall.SIGTERM:
				// stop to quit
				exitCode := executeShutdownCallbacks("SIGTERM")
				for _, f := range onProcessExit {
					f() // only perform important cleanup actions
				}
				Stop()

				os.Exit(exitCode)
			case syscall.SIGUSR1:
				// reopen
				log.Reopen()
			case syscall.SIGHUP:
				// reload
				 reconfigure()
			case syscall.SIGUSR2:
				// ignore
			}
		}
	}()
}

func catchSignalsPosix() {
	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt)

		for i := 0; true; i++ {
			<-shutdown

			if i > 0 {
				for _, f := range onProcessExit {
					f() // important cleanup actions only
				}
				os.Exit(2)
			}

			// important cleanup actions before shutdown callbacks
			for _, f := range onProcessExit {
				f()
			}

			go func() {
				os.Exit(executeShutdownCallbacks("SIGINT"))
			}()
		}
	}()
}

func executeShutdownCallbacks(signame string) (exitCode int) {
	shutdownCallbacksOnce.Do(func() {
		var errs []error

		for _, cb := range shutdownCallbacks {
			errs = append(errs, cb())
		}

		if len(errs) > 0 {
			for _, err := range errs {
				log.DefaultLogger.Printf("[ERROR] %s shutdown: %v", signame, err)
			}
			exitCode = 4
		}
	})

	return
}

func OnProcessShutDown(cb func() error) {
	shutdownCallbacks = append(shutdownCallbacks, cb)
}

func reconfigure(){
	// Stop accepting requests
	StopAccept()

	// Get socket file descriptor to pass it to fork
	listenerFD := ListListenerFD()
	if len(listenerFD) == 0 {
		log.DefaultLogger.Fatalln("no listener fd found")
	}

	log.DefaultLogger.Println("ready to pass fds:", listenerFD)

	// Set a flag for the new process start process
	os.Setenv("_MOSN_GRACEFUL_RESTART", "true")
	os.Setenv("_MOSN_INHERIT_FD", toInheritString(listenerFD))

	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}, listenerFD...),
	}

	// Fork exec the new version of your server
	fork, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.DefaultLogger.Fatalln("Fail to fork", err)
	}
	log.DefaultLogger.Println("SIGHUP received: fork-exec to", fork)

	// Wait for all conections to be finished
	WaitConnectionsDone(time.Duration(time.Second) * 15)
	log.DefaultLogger.Println(os.Getpid(), "Server gracefully shutdown")

	// Stop the old server, all the connections have been closed and the new one is running
	os.Exit(0)
}

//TODO to func, move to util package
func toInheritString(fds []uintptr) string {
	s := ""
	for _, fd := range fds {
		if len(s) > 0 {
			s += ":"
		}
		s += string(fd)
	}

	return s
}
