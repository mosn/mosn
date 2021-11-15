package transcoder

import (
	"mosn.io/mosn/pkg/log"
	goplugin "plugin"
)

type TranscoderGoPlugin struct {
	SrcPro string `json:"src_protocol,omitempty"`
	DstPro string `json:"dst_protocol,omitempty"`
	SoPath string `json:"so_path,omitempty"`
}

func (t *TranscoderGoPlugin) CreateTranscoder(listenerName string) {

	if t.SrcPro == "" || t.DstPro == "" || t.SoPath == "" {
		log.DefaultLogger.Errorf("[stream filter][transcoder] config could not be found, srcPro: %s,"+
			" dsrPro: %s, soPath: %s", t.SrcPro, t.DstPro, t.SoPath)
		return
	}

	name := listenerName + "_" + t.SrcPro + "_" + t.DstPro

	if GetTranscoder(name) != nil {
		return
	}

	p, err := goplugin.Open(t.SoPath) //根据路径加载***.so文件
	if err != nil {
		log.DefaultLogger.Errorf("[stream filter][transcoder] so file could not be load, soPath: %s, err: %v", t.SoPath, err)
		return
	}

	sym, err := p.Lookup("LoadTranscoder")
	if err != nil {
		log.DefaultLogger.Errorf("[stream filter][transcoder] so file look up error, soPath: %s, err: %v", t.SoPath, err)
		return
	}

	loadFunc := sym.(func() Transcoder)
	transcoderSo := loadFunc() //执行函数，该函数其实是初始化transcoder

	MustRegister(name, transcoderSo)
}
