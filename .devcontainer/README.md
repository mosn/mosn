# Dev Container 使用

## Dev Container 环境
- Debian GNU/Linux 11
- go version go1.14.15 linux/amd64
- [yq](https://github.com/mikefarah/yq) v1.0.1
- [just](https://github.com/casey/just/blob/master/README.%E4%B8%AD%E6%96%87.md) 1.2.0

## 制作开发镜像

如果想把自己喜欢的开发工具打进镜像里可以编辑 Dockerfile 后，重新构建自己的开发镜像，下面是制作开发镜像的步骤，
1. 进入项目的./.devcontainer路径
2. 执行 docker build -t xxx/mosn-dev:devtag -f Dockerfile .
3. （可选）执行 docker push xxx/mosn-dev:devtag 把你的镜像推送到镜像仓库

## 在[VSCode](https://code.visualstudio.com/Download)中使用
1. 在VSCode安装 [Remote - Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

2. 键盘同时按下 `Ctrl+Shift+P` (macOS 是 `Command+Shift+P`)，输入 open folder in container 选择mosn文件夹，vscode 自动拉取镜像并启动，如果你vscode的左下角看到 `Dev Container:mosn`, 就说明已经进入到Docker镜像里，接下来可以开发了