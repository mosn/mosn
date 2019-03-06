package server

import (
	. "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

/*
{
   "admin":{
	"address":{
	     "socket_address":{
		     "address": "0.0.0.0",
		     "port_value": 8888
	     }
	}
   }
}
*/
type Config interface {
	GetAdmin() *Admin
}
