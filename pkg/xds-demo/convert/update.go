// 来自原来的config/callback.go
// 所有的函数都不需要作为MOSNConfig的成员以便扩展的时候不用修改代码
// 需要注意的是 可能要实现对应的configmanager dump的逻辑
// dump逻辑改动 待定
package convert

// 有的函数甚至连*MOSNConfig作为参数都可以不要，全看实现, 没有统一的要求
func OnAddOrUpdateRouters(cfg *MOSNConfig, routers []*pb.RouteConfiguration) {
}

func OnAddOrUpdateListeners(cfg *MOSNConfig, listeners []*pb.Listener) {
}

// 其他类似
