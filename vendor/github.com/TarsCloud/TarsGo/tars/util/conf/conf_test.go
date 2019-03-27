package conf

import (
	"fmt"
	"testing"
)

func TestConf(t *testing.T) {
	f1, _ := NewConf("MMGR.TestServer.conf")
	d := f1.GetDomain("/taf/application/server")
	fmt.Println(d)
	d2 := f1.GetString("/taf/application/server<node>")
	fmt.Println(d2)
	d3 := f1.GetString("/taf/application/client/<locator>")
	fmt.Println(d3)
	d4 := f1.GetInt("/taf/application/client<sample-rate>")
	fmt.Println(d4)
	d5 := f1.GetMap("/taf/application/server/")
	fmt.Println(d5)
	fmt.Println(d5["node"])
	fmt.Println(f1.ToString())
}
