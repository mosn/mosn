package plugin

import (
	"os"

	"github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"mosn.io/mosn/pkg/plugin/proto"
)

var pluginLogBase string

func InitPlugin(log string) {
	if log != "" && log[len(log)-1] != os.PathSeparator {
		log += string(os.PathSeparator)
	}
	pluginLogBase = log
}

func GetLogPath() string {
	if pluginLogBase != "" {
		return pluginLogBase
	}
	return os.Getenv("MOSN_LOGBASE")
}

// Service is a service that Implemented by plugin main process
type Service interface {
	Call(request *proto.Request) (*proto.Response, error)
}

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "MOSN_PLUGIN",
	MagicCookieValue: "ON",
}

var PluginMap = map[string]plugin.Plugin{
	"MOSN_SERVICE": &Plugin{},
}

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type Plugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Service
}

func (p *Plugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterPluginServer(s, &server{Impl: p.Impl})
	return nil
}

func (p *Plugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &client{PluginClient: proto.NewPluginClient(c)}, nil
}
