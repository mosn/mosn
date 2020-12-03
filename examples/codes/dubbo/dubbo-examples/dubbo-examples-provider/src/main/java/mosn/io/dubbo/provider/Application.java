package mosn.io.dubbo.provider;


import mosn.io.dubbo.DemoService;
import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.config.ServiceConfig;
import org.apache.dubbo.config.spring.context.annotation.EnableDubbo;
import org.springframework.context.annotation.AnnotationConfigApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.PropertySource;

import java.util.concurrent.CountDownLatch;

public class Application {

    public static void main(String[] args) throws Exception {
        // Start Zookeeper Server
        new EmbeddedZooKeeper(2181,false).start();

        ServiceConfig<DemoService> serviceConfig = new ServiceConfig<>();
        serviceConfig.setApplication(new ApplicationConfig("dubbo-examples-provider"));
        RegistryConfig registryConfig = new RegistryConfig("zookeeper://127.0.0.1:2181");
        registryConfig.setRegister(false);
        serviceConfig.setRegistry(registryConfig);
        serviceConfig.setInterface(DemoService.class);
        serviceConfig.setRef(new DemoServiceImpl());
        serviceConfig.export();
        new CountDownLatch(1).await();
    }

}
