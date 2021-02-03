package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"mosn.io/mosn/pkg/plugin/proto"
)

type client struct {
	proto.PluginClient
}

func (c *client) Call(request *proto.Request, timeout time.Duration) (*proto.Response, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}
	response, err := c.PluginClient.Call(ctx, request)
	return response, err
}

// Client is a plugin client, It's primarily used to call request.
type Client struct {
	pclient  *plugin.Client
	config   *Config
	name     string
	fullName string
	service  *client
	enable   bool
	on       bool

	sync.Mutex
}

type Config struct {
	MaxProcs int
	Args     []string
}

func newClient(name string, config *Config) (*Client, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}
	fullName := dir + string(os.PathSeparator) + name
	if _, err := os.Stat(fullName); err != nil {
		return nil, err
	}

	c := new(Client)
	c.enable = true
	c.name = name
	c.fullName = fullName
	c.config = config
	if c.config == nil {
		c.config = &Config{1, nil}
	}

	c.Check()
	return c, nil
}

// Call invokes the function synchronously.
func (c *Client) Call(request *proto.Request, timeout time.Duration) (*proto.Response, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}
	return c.service.Call(request, timeout)
}

func (c *Client) disable() error {
	c.Lock()
	c.enable = false
	c.on = false
	client := c.pclient
	c.Unlock()

	if client != nil {
		client.Kill()
	}
	return nil
}

func (c *Client) open() error {
	c.Lock()
	c.enable = true
	c.Unlock()

	return c.Check()
}

func (c *Client) Status() (enable, on bool) {
	c.Lock()
	defer c.Unlock()

	if !c.enable {
		return c.enable, c.on
	}
	if c.pclient != nil && !c.pclient.Exited() {
		c.on = true
	} else {
		c.on = false
	}

	return c.enable, c.on
}

func (c *Client) Check() error {
	c.Lock()
	defer c.Unlock()

	if !c.enable {
		return errors.New("plugin " + c.name + " disable")
	}

	if c.pclient != nil && !c.pclient.Exited() {
		return nil
	}
	c.on = false

	cmd := exec.Command(c.fullName, c.config.Args...)

	procs := 1
	if c.config != nil && c.config.MaxProcs >= 0 && c.config.MaxProcs <= 4 {
		procs = c.config.MaxProcs
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("MOSN_PROCS=%d", procs))
	cmd.Env = append(cmd.Env, fmt.Sprintf("MOSN_LOGBASE=%s", pluginLogBase))

	pclient := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins:         PluginMap,
		Cmd:             cmd,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC},
	})

	rpcClient, err := pclient.Client()
	if err != nil {
		return err
	}

	raw, err := rpcClient.Dispense("MOSN_SERVICE")
	if err != nil {
		return err
	}

	c.pclient = pclient
	c.service = raw.(*client)

	c.on = true

	return nil
}
