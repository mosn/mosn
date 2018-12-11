package main

import (
	"fmt"
	"strings"

	envoy_config_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rbac/v2"
	"github.com/gogo/protobuf/jsonpb"
)

func pb2json(rbac *envoy_config_v2.RBAC) (string, error) {
	var m jsonpb.Marshaler
	if js, err := m.MarshalToString(rbac); err != nil {
		return "", err
	} else {
		return js, nil
	}
}

func json2pb(jsonString string) (*envoy_config_v2.RBAC, error) {
	var un jsonpb.Unmarshaler
	rbac := new(envoy_config_v2.RBAC)
	if err := un.Unmarshal(strings.NewReader(jsonString), rbac); err != nil {
		return nil, err
	} else {
		return rbac, nil
	}
}

func main() {
	jsonString, err := pb2json(rbacExp)
	if err != nil {
		fmt.Printf("pb2json error: %v", err)
	} else {
		fmt.Println(jsonString)
	}

	if rbac, err := json2pb(jsonString); err != nil {
		fmt.Printf("json2pb error: %v", err)
	} else {
		fmt.Println(rbac)
	}
}
