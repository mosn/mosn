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

package basic

import "github.com/alipay/sofa-mosn/pkg/types"

type RouteRuleImplAdaptor struct {
}

func (r *RouteRuleImplAdaptor) ClusterName() string {
	return ""
}

func (r *RouteRuleImplAdaptor) Priority() types.Priority {
	return types.PriorityDefault
}

func (r *RouteRuleImplAdaptor) VirtualCluster(headers map[string]string) types.VirtualCluster {
	return nil
}

func (r *RouteRuleImplAdaptor) Policy() types.Policy {
	return nil
}

//func (r *RouteRuleImplAdaptor) MetadataMatcher() types.MetadataMatcher {
//	return nil
//}

func (r *RouteRuleImplAdaptor) VirtualHost() types.VirtualHost {
	return nil
}

func (r *RouteRuleImplAdaptor) Metadata() types.RouteMetaData {
	return nil
}

func (r *RouteRuleImplAdaptor) MetadataMatchCriteria(clusterName string) types.MetadataMatchCriteria {
	return nil
}
