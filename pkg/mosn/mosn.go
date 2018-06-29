package main

import (
	_ "flag"
	"github.com/urfave/cli"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/healthcheck"
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/xds"
	_ "net/http/pprof"
	"os"
	"time"
)

func main() {
	app := cli.NewApp()
	app.Name = "mosn"
	app.Version = "0.0.1"
	app.Compiled = time.Now()
	app.Copyright = "(c) 2018 Ant Financial"
	app.Usage = "MOSN is modular observable smart netstub."

	//commands
	app.Commands = []cli.Command{
		cmdStart,
		cmdStop,
		cmdReload,
	}

	//action
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)

		c.App.Setup()
		return nil
	}

	// ignore error so we don't exit non-zero and break gfmrun README example tests
	_ = app.Run(os.Args)
}
