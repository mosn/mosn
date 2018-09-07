## 1 常见问题列表
___

### 1.1 Unsupported protocol jsonrpc
-

问题描述: 用java写的dubbo provider端程序如果遇到下面的错误：

 java.lang.IllegalStateException: Unsupported protocol jsonrpc in notified url:   
 jsonrpc://116.211.15.189:8080/im.ikurento.user.UserProvider?anyhost=true&application=token-dubbo-p&default.threads=100&dubbo=2.5.3...

 from registry 116.211.15.190:2181 to consumer 10.241.19.54, supported protocol: [dubbo, http, injvm, mock, redis, registry, rmi, thrift]

错误原因：provider端声明使用了jsonrpc，所以所有的协议都默认支持了jsonrpc协议。

解决方法：服务需要在dubbo.provider.xml中明确支持dubbo协议，请在reference中添加protocol="dubbo"，如：

	<dubbo:protocol name="dubbo" port="28881" />
	<dubbo:protocol name="jsonrpc" port="38881" server="jetty" />
	<dubbo:service id="userService" interface="im.ikurento.user.UserProvider" check="false" timeout="5000" protocol="dubbo"/>
 	<dubbo:service id="userService" interface="im.ikurento.user.UserProvider" check="false" timeout="5000" protocol="jsonrpc"/>

与本问题无关，补充一些消费端的配置：

	<dubbo:reference id="userService" interface="im.ikurento.user.UserService" connections="2" check="false">
		<dubbo:method name="GetUser" async="true" return="false" />
 	</dubbo:reference>

## 2 配置文件
dubbogo client端根据其配置文件client.toml的serviceList来watch zookeeper上相关服务，否则当相关服务的zk provider path下的node发生增删的时候，因为关注的service不正确而导致不能收到相应的通知。

dubbogo/selector/get接口会通过dubbogo/registry/watch:GetServiceList接口来获取到相应服务的provider列表。

所以务必注意这个配置文件中的serviceList与实际代码中相关类的Service函数提供的Service一致。

## 3 其他注意事项
- a. dubbo 配置多个 zk 地址；
- b. 消费者在 dubbo 的配置文件中相关interface配置务必指定protocol, 如protocol="dubbo";
- c. java dubbo provider提供服务的method如果要提供给dubbogo client服务，则method的首字母必须是大写;