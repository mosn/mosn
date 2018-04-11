package main

import (
	"github.com/urfave/cli"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
)

var (
	cmdStart = cli.Command{
		Name:  "start",
		Usage: "start mosn proxy",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "config, c",
				Usage:  "Load configuration from `FILE`",
				EnvVar: "MOSN_CONFIG",
				Value:  "config/mson.conf",
			},
		},
		Action: func(c *cli.Context) error {
			configPath := c.String("config")
			conf := config.Load(configPath)
			Start(conf)
			return nil
		},
	}

	cmdStop = cli.Command{
		Name:  "stop",
		Usage: "stop mosn proxy",
		Action: func(c *cli.Context) error {
			return nil
		},
	}

	cmdReload = cli.Command{
		Name:  "reload",
		Usage: "reconfiguration",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
)
