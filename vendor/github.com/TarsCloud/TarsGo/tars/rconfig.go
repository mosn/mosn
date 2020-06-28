package tars

import (
	"fmt"
	"io/ioutil"

	"github.com/TarsCloud/TarsGo/tars/protocol/res/configf"
)

func saveFile(path string, filename string, content string) error {
	err := ioutil.WriteFile(fmt.Sprintf("%s/%s", path, filename), []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}

//RConf struct for geting remote config.
type RConf struct {
	app    string
	server string
	comm   *Communicator
	tc     *configf.Config
	path   string
}

//NewRConf init a Rconf, path should be getting from GetServerConfig().BasePath
func NewRConf(app string, server string, path string) *RConf {
	comm := NewCommunicator()
	tc := new(configf.Config)
	obj := GetServerConfig().config

	comm.StringToProxy(obj, tc)
	return &RConf{app, server, comm, tc, path}
}

//GetConfigList is discarded.
func (c *RConf) GetConfigList() (flist []string, err error) {
	info := configf.GetConfigListInfo{
		Appname:    c.app,
		Servername: c.server,
		/*
		   Host:string
		   Setdivision:string
		   Containername:string
		*/
	}
	ret, err := c.tc.ListAllConfigByInfo(&info, &flist)
	if err != nil {
		return flist, err
	}
	if ret != 0 {
		return flist, fmt.Errorf("ret:%d", ret)
	}
	return flist, nil
}

//GetConfig gets the remote config and save it to the path, also return the content.
func (c *RConf) GetConfig(filename string) (config string, err error) {
	var set string
	if v, ok := c.comm.GetProperty("setdivision"); ok {
		set = v
	}
	info := configf.ConfigInfo{
		Appname:     c.app,
		Servername:  c.server,
		Filename:    filename,
		Setdivision: set,
	}
	ret, err := c.tc.LoadConfigByInfo(&info, &config)
	if err != nil {
		return config, err
	}
	if ret != 0 {
		return config, fmt.Errorf("ret %d", ret)
	}
	err = saveFile(c.path, filename, config)
	if err != nil {
		return config, err
	}
	return config, nil
}
