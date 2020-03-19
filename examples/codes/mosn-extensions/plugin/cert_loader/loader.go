package main

import (
	"encoding/json"
	"io/ioutil"

	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
)

type cert struct {
	CertPem string `json:"cert_pem"`
	KeyPem  string `json:"key_pem"`
	CAPem   string `json:"ca_pem"`
}

var demoCertData []byte

type loader struct{}

func (l *loader) Call(request *proto.Request) (*proto.Response, error) {
	response := new(proto.Response)
	response.Body = demoCertData
	return response, nil
}

func demo() {
	certdata, _ := ioutil.ReadFile("./certs/cert.pem")
	keydata, _ := ioutil.ReadFile("./certs/key.pem")
	cadata, _ := ioutil.ReadFile("./certs/ca.pem")
	demoCert := &cert{
		CertPem: string(certdata),
		KeyPem:  string(keydata),
		CAPem:   string(cadata),
	}
	b, _ := json.Marshal(demoCert)
	demoCertData = b
}

func main() {
	demo()
	plugin.Serve(&loader{})
}
