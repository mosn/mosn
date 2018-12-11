package main

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_config_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rbac/v2"
	. "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2alpha"
	"github.com/envoyproxy/go-control-plane/envoy/type/matcher"
)

var rbacExp = &envoy_config_v2.RBAC{
	Rules: &RBAC{
		Action: RBAC_ALLOW,
		Policies: map[string]*Policy{
			"service-admin": {
				Permissions: []*Permission{
					{
						Rule: &Permission_Any{Any: true},
					},
				},
				Principals: []*Principal{
					{
						Identifier: &Principal_Authenticated_{
							Authenticated: &Principal_Authenticated{
								PrincipalName: &matcher.StringMatcher{
									MatchPattern: &matcher.StringMatcher_Exact{Exact: "cluster.local/ns/default/sa/admin"},
								},
							},
						},
					},
					{
						Identifier: &Principal_Authenticated_{
							Authenticated: &Principal_Authenticated{
								PrincipalName: &matcher.StringMatcher{
									MatchPattern: &matcher.StringMatcher_Exact{Exact: "cluster.local/ns/default/sa/superuser"},
								},
							},
						},
					},
				},
			},
		},
	},
	ShadowRules: &RBAC{
		Action: RBAC_DENY,
		Policies: map[string]*Policy{
			"product-viewer": {
				Permissions: []*Permission{
					{
						Rule: &Permission_AndRules{
							AndRules: &Permission_Set{
								Rules: []*Permission{
									{
										Rule: &Permission_Header{
											Header: &route.HeaderMatcher{
												Name: ":method",
												HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
													ExactMatch: "GET",
												},
											},
										},
									},
									{
										Rule: &Permission_Header{
											Header: &route.HeaderMatcher{
												Name: ":path",
												HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
													ExactMatch: "/products(/.*)?",
												},
											},
										},
									},
									{
										Rule: &Permission_OrRules{
											OrRules: &Permission_Set{
												Rules: []*Permission{
													{
														Rule: &Permission_DestinationPort{DestinationPort: 80},
													},
													{
														Rule: &Permission_DestinationPort{DestinationPort: 443},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Principals: []*Principal{
					{
						Identifier: &Principal_Any{Any: true},
					},
				},
			},
		},
	},
}
