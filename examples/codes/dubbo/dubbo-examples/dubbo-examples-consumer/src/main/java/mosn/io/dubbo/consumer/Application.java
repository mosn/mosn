package mosn.io.dubbo.consumer;

import mosn.io.dubbo.DemoService;


import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.config.spring.context.annotation.EnableDubbo;
import org.springframework.context.annotation.AnnotationConfigApplicationContext;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.PropertySource;

public class Application {
    /**
     * In order to make sure multicast registry works, need to specify '-Djava.net.preferIPv4Stack=true' before
     * launch the application
     */
    public static void main(String[] args) {
        ReferenceConfig<DemoService> referenceConfig = new ReferenceConfig<>();
        referenceConfig.setInterface(DemoService.class);
        referenceConfig.setUrl("dubbo://127.0.0.1:2045");
        ApplicationConfig applicationConfig = new ApplicationConfig("dubbo-examples-consumer");
        applicationConfig.setQosEnable(false);
        referenceConfig.setApplication(applicationConfig);
        DemoService demoService = referenceConfig.get();
        while (true){
            try {
                System.out.println(demoService.sayHello("MOSN"));
                Thread.sleep(5000);
            } catch (Exception e) {
                e.printStackTrace();
            }
        }
    }

}
