package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-plugin"
	"sofastack.io/sofa-mosn/pkg/log"
)

func main() {
	lg := &log.Logger{}
	file, err := os.OpenFile("/tmp/plugin.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Print(err)
		return
	}
	lg.SetWriter(file)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "log",
			MagicCookieValue: "log",
		},
		Plugins: map[string]plugin.Plugin{
			"log": lg,
		},
	})
}