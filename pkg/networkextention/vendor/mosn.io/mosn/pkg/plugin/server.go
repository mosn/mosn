package plugin

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"mosn.io/mosn/pkg/plugin/proto"
)

var parentPid int

func init() {
	parentPid = os.Getppid()
}

type server struct {
	// This s the real implementation
	Impl Service
}

func (s *server) Call(ctx context.Context, req *proto.Request) (*proto.Response, error) {
	return s.Impl.Call(req)
}

func checkParentAlive() {
	go func() {
		for {
			if parentPid == 1 || os.Getppid() != parentPid {
				fmt.Println("parent no alive, exit")
				os.Exit(0)
			}
			_, err := os.FindProcess(parentPid)
			if err != nil {
				fmt.Println("parent no alive, exit")
				os.Exit(0)
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

// Serve is a function used to serve a plugin. This should be ran on the plugin's main process.
func Serve(service Service) {
	checkParentAlive()

	p := os.Getenv("MOSN_PROCS")
	if procs, err := strconv.Atoi(p); err == nil {
		runtime.GOMAXPROCS(procs)
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"MOSN_SERVICE": &Plugin{Impl: service},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
