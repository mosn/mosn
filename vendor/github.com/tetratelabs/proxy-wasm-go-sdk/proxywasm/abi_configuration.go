// Copyright 2020 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxywasm

//export proxy_on_vm_start
func proxyOnVMStart(rootContextID uint32, vmConfigurationSize int) bool {
	ctx, ok := currentState.rootContexts[rootContextID]
	if !ok {
		panic("invalid context on proxy_on_vm_start")
	}
	currentState.setActiveContextID(rootContextID)
	return ctx.context.OnVMStart(vmConfigurationSize)
}

//export proxy_on_configure
func proxyOnConfigure(rootContextID uint32, pluginConfigurationSize int) bool {
	ctx, ok := currentState.rootContexts[rootContextID]
	if !ok {
		panic("invalid context on proxy_on_configure")
	}
	currentState.setActiveContextID(rootContextID)
	return ctx.context.OnPluginStart(pluginConfigurationSize)
}
