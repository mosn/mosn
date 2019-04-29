// Package rconf implements fetch tars config and save it.
//
// 由于平台正在迁移tconf，这个库仅仅是为了兼容而存在，所以功能上非常少.
//
// Tconf的golang库正在开发中.
package tars

import (
	"errors"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/configf"
	"io/ioutil"
)

func saveFile(path string, filename string, content string) error {
	err := ioutil.WriteFile(fmt.Sprintf("%s/%s", path, filename), []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}

type RConf struct {
	app    string
	server string
	comm   *Communicator
	tc     *configf.Config
	path   string
}

//创建一个RConf对象，path目录最好从GetServerConfig().BasePath里取
func NewRConf(app string, server string, path string) *RConf {
	comm := NewCommunicator()
	tc := new(configf.Config)
	obj := "tars.tarsconfig.ConfigObj"

	comm.StringToProxy(obj, tc)
	return &RConf{app, server, comm, tc, path}
}

//这个API可能是TARS废弃的，请勿使用
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
		return flist, errors.New(fmt.Sprintln("ret:%d", ret))
	}
	return flist, nil
}

//从远程拉取配置文件并保存到path目录里。
//
//同时将配置文件内容返回。
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
		return config, errors.New(fmt.Sprintln("ret:%d", ret))
	}
	err = saveFile(c.path, filename, config)
	if err != nil {
		return config, err
	}
	return config, nil
}
