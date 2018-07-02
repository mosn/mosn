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
				Value:  "resource/mosn_config.json",
			}, cli.StringFlag{
				Name:   "service-cluster, s",
				Usage:  "sidecar service cluster",
				EnvVar: "SERVICE_CLUSTER",
			}, cli.StringFlag{
				Name:   "service-node, n",
				Usage:  "sidecar service node",
				EnvVar: "SERVICE_NODE",
			},
		},
		Action: func(c *cli.Context) error {
			configPath := c.String("config")
			serviceCluster := c.String("service-cluster")
			serviceNode := c.String("service-node")
			conf := config.Load(configPath)
			Start(conf, serviceCluster, serviceNode)
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
