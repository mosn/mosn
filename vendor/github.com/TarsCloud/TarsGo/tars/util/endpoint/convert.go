package endpoint

import "github.com/TarsCloud/TarsGo/tars/protocol/res/endpointf"

func Tars2endpoint(end endpointf.EndpointF) Endpoint {
	proto := "tcp"
	if end.Istcp == 0 {
		proto = "udp"
	}
	return Endpoint{
		Host:    end.Host,
		Port:    int32(end.Port),
		Timeout: int32(end.Timeout),
		Istcp:   end.Istcp,
		Proto:   proto,
		Bind:    "",
		//Container: end.ContainerName,
		SetId: end.SetId,
	}

}

func Endpoint2tars(end Endpoint) endpointf.EndpointF {
	return endpointf.EndpointF{
		Host:    end.Host,
		Port:    int32(end.Port),
		Timeout: int32(end.Timeout),
		Istcp:   end.Istcp,
		//	ContainerName: end.Container,
		SetId: end.SetId,
	}
}
