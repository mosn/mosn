# 快速开始

本文用于帮助初次接触 MOSN 项目的开发人员，快速搭建开发环境，完成构建，测试和打包。
MOSN 基于 Golang 1.9.2 研发，使用dep进行依赖管理

## 准备运行环境

+ 如果你使用容器运行MOSN, 请先 [安装docker](https://docs.docker.com/install/)
+ 如果你使用本地机器，请使用类unix环境
+ 安装 go 的编译环境 
+ 安装 dep : 参考[官方安装文档](https://golang.github.io/dep/docs/installation.html)

## 获取代码

MOSN 项目的代码托管在 [github](https://github.com/alipay/sofa-mosn)，获取方式如下：

```bash
go get github.com/alipay/sofa-mosn
```

如果你的 go get 下载存在问题，请手动创建项目工程

```bash
# 进入GOPATH下的src目录
cd $GOPATH/src
# 创建 github.com/alipay 目录
mkdir -p github.com/alipay
cd github.com/alipay

# clone mosn代码
git clone git@github.com:alipay/sofa-mosn.git
cd sofa-mosn
```

最终MOSN的源代码代码路径为 `$GOPATH/src/github.com/alipay/sofa-mosn`

## 导入IDE

使用您喜爱的Golang IDE导入 `$GOPATH/src/github.com/alipay/sofa-mosn` 项目，推荐Goland。

## 编译代码

在项目根目录下，根据自己机器的类型以及欲执行二进制的环境，选择以下命令编译 MOSN 的二进制文件：
+ 使用 docker 镜像编译
```bash
    make build // 编译出 linux 64bit 可运行二进制文件
```
+ 非 docker，本地编译
    + 编译本地可运行二进制文件
    ```bash
        make build-local
    ```
    + 非 linux 机器交叉编译 linux 64bit 可运行二进制文件
    ```bash
        make build-linux64
    ```
    + 非 linux 机器交叉编译 linux 32bit 可运行二进制文件
    ```bash
        make build-linux32
    ```
完成后可以在 `build/bundles/${version}/binary` 目录下找到编译好的二进制文件。

## 打包

+ 在项目根目录下执行如下命令进行打包：

```bash
make rpm
```

完成后可以在 `build/bundles/${version}/rpm` 目录下找到打包好的文件。


## 运行测试

在项目根目录下执行如下命令运行单元测试：

```bash
make unit-test
```

单独运行 MOSN 作为 proxy 转发的示例:

+ 参考 `sofa-mosn/test/` 下的[示例](testandsamples/RunMosnTests.md)

## 从配置文件[启动 MOSN](../reference/HowtoStartMosnFromConfig.md)

```bash
 mosn start -c '$CONFIG_FILE'
```

## 如何快速启动一个 mosn 的转发程序

参考 `examples` 目录下的示例工程

+ [以sofa proxy为例](testandsamples/RunMosnSofaProxy.md)
+ [以http proxy为例](testandsamples/RunMosnHttpProxy.md)
