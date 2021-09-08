package server

import "mosn.io/mosn/pkg/config/v2"

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
	GetAdmin() *v2.Admin
}
