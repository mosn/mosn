package v1

import (

	"fmt"
	"io/ioutil"
	"encoding/json"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/gogo/protobuf/jsonpb"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func (c *xdsClient) getListeners(endpoint string) Listeners{
	url := c.getLDSResquest(endpoint)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		log.DefaultLogger.Errorf("couldn't get listeners: %v", err)
		//fmt.Printf("couldn't get listeners: %v\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.DefaultLogger.Errorf("read body error when get listeners: %v", err)
		//fmt.Printf("read body error when get listeners: %v\n", err)
		return nil
	}

	return c.parseListeners(body)
}

func (c *xdsClient) parseListeners(body []byte) Listeners {
	var res ListenersData

	err := json.Unmarshal(body, &res)
	if err != nil {
		log.DefaultLogger.Errorf("read body error when get listeners: %v", err)
		//fmt.Printf("read body error when get listeners: %v\n", err)
		return nil
	}

	listeners := make([]*xdsapi.Listener, 0, len(res.ListenersV1))
	for _, listenerV1 := range res.ListenersV1 {
		listenerV2 := &xdsapi.Listener{}
		filterChain := envoy_api_v2_listener.FilterChain{}

		listenerV2.Name = listenerV1.Name
		err := translateAddress(listenerV1.Address, true, true, &listenerV2.Address)
		if err != nil {
			log.DefaultLogger.Errorf("fail to translate address : %v\n", err)
			//fmt.Printf("fail to translate address : %v\n", err)
			return nil
		}

		listenerV2.UseOriginalDst = &google_protobuf.BoolValue{listenerV1.UseOriginalDst}
		listenerV2.DeprecatedV1 = &xdsapi.Listener_DeprecatedV1{}
		listenerV2.DeprecatedV1.BindToPort = &google_protobuf.BoolValue{listenerV1.BindToPort}
		listenerV2.PerConnectionBufferLimitBytes = &google_protobuf.UInt32Value{uint32(listenerV1.PerConnectionBufferLimitBytes)}
		if listenerV1.DrainType == "modify_only" {
			listenerV2.DrainType = xdsapi.Listener_MODIFY_ONLY
		}

		filterChain.UseProxyProto = &google_protobuf.BoolValue{listenerV1.UseProxyProto}
		//filterChain.TlsContext = translateDownstreamTlsContext(listenerV1.SSLContext)

		filterChain.Filters = make([]envoy_api_v2_listener.Filter, 0, len(listenerV1.Filters))
		for _, filterV1 := range listenerV1.Filters {
			filterV2 := envoy_api_v2_listener.Filter{}
			filterV2.Name = filterV1.Name
			filterV2.DeprecatedV1 = &envoy_api_v2_listener.Filter_DeprecatedV1{}
			filterV2.DeprecatedV1.Type = filterV1.Type
			filterV2.Config = &google_protobuf.Struct{}

			filterConfig := fmt.Sprintf("{\"deprecated_v1\": true, \"value\": %s}", string(filterV1.Config))
			err = jsonpb.UnmarshalString(filterConfig, filterV2.Config)
			if err != nil {
				log.DefaultLogger.Errorf("fail to translate filter config : %v\n", err)
				//fmt.Printf("fail to translate filter config : %v\n", err)
				return nil
			}
			filterChain.Filters = append(filterChain.Filters, filterV2)
		}

		listenerV2.FilterChains = make([]envoy_api_v2_listener.FilterChain, 0, 1)
		listenerV2.FilterChains = append(listenerV2.FilterChains, filterChain)
		listeners = append(listeners, listenerV2)
	}

	return listeners
}

func (c *xdsClient) getLDSResquest(endpoint string) string {
	return fmt.Sprintf("http://%s/v1/listeners/%s/%s", endpoint, c.serviceCluster, c.serviceNode)
}