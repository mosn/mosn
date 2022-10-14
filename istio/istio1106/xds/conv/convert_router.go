/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package conv

import (
	"net/http"
	"strings"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3" // some config contains this protobuf, mosn does not parse it yet.

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/any"
	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/log"
)

func ConvertRouterConf(routeConfigName string, xdsRouteConfig *envoy_config_route_v3.RouteConfiguration) (*v2.RouterConfiguration, bool) {
	if routeConfigName != "" {
		return &v2.RouterConfiguration{
			RouterConfigurationConfig: v2.RouterConfigurationConfig{
				RouterConfigName: routeConfigName,
			},
		}, true
	}

	if xdsRouteConfig == nil {
		return nil, false
	}

	virtualHosts := make([]v2.VirtualHost, 0)

	for _, xdsVirtualHost := range xdsRouteConfig.GetVirtualHosts() {
		virtualHost := v2.VirtualHost{
			Name:    xdsVirtualHost.GetName(),
			Domains: xdsVirtualHost.GetDomains(),
			Routers: convertRoutes(xdsVirtualHost.GetRoutes()),
			//RequireTLS:              xdsVirtualHost.GetRequireTls().String(),
			//VirtualClusters:         convertVirtualClusters(xdsVirtualHost.GetVirtualClusters()),
			RequestHeadersToAdd:     convertHeadersToAdd(xdsVirtualHost.GetRequestHeadersToAdd()),
			ResponseHeadersToAdd:    convertHeadersToAdd(xdsVirtualHost.GetResponseHeadersToAdd()),
			ResponseHeadersToRemove: xdsVirtualHost.GetResponseHeadersToRemove(),
		}
		virtualHosts = append(virtualHosts, virtualHost)
	}

	return &v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName:        xdsRouteConfig.GetName(),
			RequestHeadersToAdd:     convertHeadersToAdd(xdsRouteConfig.GetRequestHeadersToAdd()),
			ResponseHeadersToAdd:    convertHeadersToAdd(xdsRouteConfig.GetResponseHeadersToAdd()),
			ResponseHeadersToRemove: xdsRouteConfig.GetResponseHeadersToRemove(),
		},
		VirtualHosts: virtualHosts,
	}, false
}

func convertRoutes(xdsRoutes []*envoy_config_route_v3.Route) []v2.Router {
	if xdsRoutes == nil {
		return nil
	}
	routes := make([]v2.Router, 0, len(xdsRoutes))
	for _, xdsRoute := range xdsRoutes {
		if xdsRouteAction := xdsRoute.GetRoute(); xdsRouteAction != nil {
			route := v2.Router{
				RouterConfig: v2.RouterConfig{
					Match: convertRouteMatch(xdsRoute.GetMatch()),
					Route: convertRouteAction(xdsRouteAction),
					//Decorator: v2.Decorator(xdsRoute.GetDecorator().String()),
					RequestMirrorPolicies: convertMirrorPolicy(xdsRouteAction),
				},
				Metadata: convertMeta(xdsRoute.GetMetadata()),
			}
			route.PerFilterConfig = convertPerRouteConfig(xdsRoute.GetTypedPerFilterConfig())
			routes = append(routes, route)
		} else if xdsRouteAction := xdsRoute.GetRedirect(); xdsRouteAction != nil {
			route := v2.Router{
				RouterConfig: v2.RouterConfig{
					Match:    convertRouteMatch(xdsRoute.GetMatch()),
					Redirect: convertRedirectAction(xdsRouteAction),
					//Decorator: v2.Decorator(xdsRoute.GetDecorator().String()),
				},
				Metadata: convertMeta(xdsRoute.GetMetadata()),
			}
			route.PerFilterConfig = convertPerRouteConfig(xdsRoute.GetTypedPerFilterConfig())
			routes = append(routes, route)
		} else if xdsRouteAction := xdsRoute.GetDirectResponse(); xdsRouteAction != nil {
			route := v2.Router{
				RouterConfig: v2.RouterConfig{
					Match:          convertRouteMatch(xdsRoute.GetMatch()),
					DirectResponse: convertDirectResponseAction(xdsRouteAction),
					//Decorator: v2.Decorator(xdsRoute.GetDecorator().String()),
				},
				Metadata: convertMeta(xdsRoute.GetMetadata()),
			}
			route.PerFilterConfig = convertPerRouteConfig(xdsRoute.GetTypedPerFilterConfig())
			routes = append(routes, route)
		} else {
			log.DefaultLogger.Errorf("unsupported route actin, just Route, Redirect and DirectResponse support yet, ignore this route")
			continue
		}
	}
	return routes
}

func convertPerRouteConfig(xdsPerRouteConfig map[string]*any.Any) map[string]interface{} {
	perRouteConfig := make(map[string]interface{}, 0)

	for key, config := range xdsPerRouteConfig {
		switch key {
		case v2.FaultStream, wellknown.Fault:
			cfg, err := convertStreamFaultInjectConfig(config)
			if err != nil {
				log.DefaultLogger.Infof("convertPerRouteConfig[%s] error: %v", v2.FaultStream, err)
				continue
			}
			log.DefaultLogger.Debugf("add a fault inject stream filter in router")
			perRouteConfig[v2.FaultStream] = cfg
		case v2.PayloadLimit:
			if featuregate.Enabled(featuregate.PayLoadLimitEnable) {
				//cfg, err := convertStreamPayloadLimitConfig(config)
				//if err != nil {
				//      log.DefaultLogger.Infof("convertPerRouteConfig[%s] error: %v", v2.PayloadLimit, err)
				//      continue
				//}
				//log.DefaultLogger.Debugf("add a payload limit stream filter in router")
				//perRouteConfig[v2.PayloadLimit] = cfg
			}
		default:
			log.DefaultLogger.Warnf("unknown per route config: %s", key)
		}
	}

	return perRouteConfig
}

func convertRouteMatch(xdsRouteMatch *envoy_config_route_v3.RouteMatch) v2.RouterMatch {
	rm := v2.RouterMatch{
		Prefix: xdsRouteMatch.GetPrefix(),
		Path:   xdsRouteMatch.GetPath(),
		//CaseSensitive: xdsRouteMatch.GetCaseSensitive().GetValue(),
		//Runtime:       convertRuntime(xdsRouteMatch.GetRuntime()),
		Headers: convertHeaders(xdsRouteMatch.GetHeaders()),
	}
	if xdsRouteMatch.GetSafeRegex() != nil {
		rm.Regex = xdsRouteMatch.GetSafeRegex().Regex
	}
	return rm
}

func convertHeaders(xdsHeaders []*envoy_config_route_v3.HeaderMatcher) []v2.HeaderMatcher {
	if xdsHeaders == nil {
		return nil
	}
	headerMatchers := make([]v2.HeaderMatcher, 0, len(xdsHeaders))
	for _, xdsHeader := range xdsHeaders {
		headerMatcher := v2.HeaderMatcher{}
		if xdsHeader.GetSafeRegexMatch() != nil && xdsHeader.GetSafeRegexMatch().Regex != "" {
			headerMatcher.Name = xdsHeader.GetName()
			headerMatcher.Value = xdsHeader.GetSafeRegexMatch().Regex
			headerMatcher.Regex = true
		} else {
			headerMatcher.Name = xdsHeader.GetName()
			headerMatcher.Value = xdsHeader.GetExactMatch()
			headerMatcher.Regex = false
		}

		// as pseudo headers not support when Http1.x upgrade to Http2, change pseudo headers to normal headers
		// this would be fix soon
		if strings.HasPrefix(headerMatcher.Name, ":") {
			headerMatcher.Name = headerMatcher.Name[1:]
		}
		headerMatchers = append(headerMatchers, headerMatcher)
	}
	return headerMatchers
}

func convertMeta(xdsMeta *envoy_config_core_v3.Metadata) api.Metadata {
	if xdsMeta == nil {
		return nil
	}
	meta := make(map[string]string, len(xdsMeta.GetFilterMetadata()))
	for key, value := range xdsMeta.GetFilterMetadata() {
		if key == "envoy.lb" {
			for eKey, eValue := range value.Fields {
				meta[eKey] = eValue.GetStringValue()
			}
		}
		meta[key] = value.String()
	}
	return meta
}

func convertRouteAction(xdsRouteAction *envoy_config_route_v3.RouteAction) v2.RouteAction {
	if xdsRouteAction == nil {
		return v2.RouteAction{}
	}
	return v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName:      xdsRouteAction.GetCluster(),
			ClusterHeader:    xdsRouteAction.GetClusterHeader(),
			WeightedClusters: convertWeightedClusters(xdsRouteAction.GetWeightedClusters()),
			HashPolicy:       convertHashPolicy(xdsRouteAction.GetHashPolicy()),
			RetryPolicy:      convertRetryPolicy(xdsRouteAction.GetRetryPolicy()),
			PrefixRewrite:    xdsRouteAction.GetPrefixRewrite(),
			AutoHostRewrite:  xdsRouteAction.GetAutoHostRewrite().GetValue(),
			//RequestHeadersToAdd:     convertHeadersToAdd(xdsRouteAction.GetRequestHeadersToAdd()),
			//
			//ResponseHeadersToAdd:    convertHeadersToAdd(xdsRouteAction.GetResponseHeadersToAdd()),
			//ResponseHeadersToRemove: xdsRouteAction.GetResponseHeadersToRemove(),
		},
		MetadataMatch: convertMeta(xdsRouteAction.GetMetadataMatch()),
		Timeout:       ConvertDuration(xdsRouteAction.GetTimeout()),
	}
}

func convertHeadersToAdd(headerValueOption []*envoy_config_core_v3.HeaderValueOption) []*v2.HeaderValueOption {
	if len(headerValueOption) < 1 {
		return nil
	}
	valueOptions := make([]*v2.HeaderValueOption, 0, len(headerValueOption))
	for _, opt := range headerValueOption {
		var isAppend *bool
		if opt.Append != nil {
			appendVal := opt.GetAppend().GetValue()
			isAppend = &appendVal
		}
		valueOptions = append(valueOptions, &v2.HeaderValueOption{
			Header: &v2.HeaderValue{
				Key:   opt.GetHeader().GetKey(),
				Value: opt.GetHeader().GetValue(),
			},
			Append: isAppend,
		})
	}
	return valueOptions
}

func convertRetryPolicy(xdsRetryPolicy *envoy_config_route_v3.RetryPolicy) *v2.RetryPolicy {
	if xdsRetryPolicy == nil {
		return &v2.RetryPolicy{}
	}
	return &v2.RetryPolicy{
		RetryPolicyConfig: v2.RetryPolicyConfig{
			RetryOn:    len(xdsRetryPolicy.GetRetryOn()) > 0,
			NumRetries: xdsRetryPolicy.GetNumRetries().GetValue(),
		},
		RetryTimeout: ConvertDuration(xdsRetryPolicy.GetPerTryTimeout()),
	}
}

func convertRedirectAction(xdsRedirectAction *envoy_config_route_v3.RedirectAction) *v2.RedirectAction {
	if xdsRedirectAction == nil {
		return nil
	}
	return &v2.RedirectAction{
		SchemeRedirect: xdsRedirectAction.GetSchemeRedirect(),
		HostRedirect:   xdsRedirectAction.GetHostRedirect(),
		PathRedirect:   xdsRedirectAction.GetPathRedirect(),
		ResponseCode:   convertRedirectResponseCode(xdsRedirectAction.GetResponseCode()),
	}
}

func convertRedirectResponseCode(responseCode envoy_config_route_v3.RedirectAction_RedirectResponseCode) int {
	switch responseCode {
	case envoy_config_route_v3.RedirectAction_MOVED_PERMANENTLY:
		return http.StatusMovedPermanently
	case envoy_config_route_v3.RedirectAction_FOUND:
		return http.StatusFound
	case envoy_config_route_v3.RedirectAction_SEE_OTHER:
		return http.StatusSeeOther
	case envoy_config_route_v3.RedirectAction_TEMPORARY_REDIRECT:
		return http.StatusTemporaryRedirect
	case envoy_config_route_v3.RedirectAction_PERMANENT_REDIRECT:
		return http.StatusPermanentRedirect
	default:
		return http.StatusMovedPermanently
	}
}

func convertDirectResponseAction(xdsDirectResponseAction *envoy_config_route_v3.DirectResponseAction) *v2.DirectResponseAction {
	if xdsDirectResponseAction == nil {
		return nil
	}

	var body string
	if rawData := xdsDirectResponseAction.GetBody(); rawData != nil {
		body = rawData.GetInlineString()
	}

	return &v2.DirectResponseAction{
		StatusCode: int(xdsDirectResponseAction.GetStatus()),
		Body:       body,
	}
}
