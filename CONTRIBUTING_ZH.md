# MOSN 贡献者指南

MOSN 基于 Apache 2.0 许可发布，遵循标准的 Github 开发流程，使用 Github Issue 跟踪问题并将 Pull Request 合并到 master 分支中。本文档可以帮助你了解如何参与贡献。

在提交 Pull Request 之前您需要先签署[贡献者许可协议（CLA）](http://cla.sofastack.tech/)。虽然签署 CLA 不会给您授予主存储库的提交权利，但这意味着我们可以接受您的贡献，签署 CLA 后您将成为本项目的荣誉作者。对于活跃的贡献者甚至可以加入我们的核心团队，并具有合并 Pull Request 的权限。

## 代码约定

以下对于 Pull Request 的要求并非强制，但是会对您提交 Pull Request 有帮助。

1. 代码格式
  - 命令行方式：在命令行中运行 `goimports -w yourfile.go` 或者 `golint yourfile.go` 格式化代码
  - IDE 方式：使用如 Goland IDE，在 Go->imports 页面中选择 ”Group stdlib imports“ - “Move all stdlib imports in a single group” - “Move all imports in a single declaration”   
  - 在运行 CI 测试的时候会检查代码格式，请确保您在提交代码之前已对代码进行格式化
2. 确保所有新的 `.go` 文件都具有简单的 doc 类注释，其中至少包含一个 `author` 标记标识您，并且最好至少包含一个有关该类用途的段落。
3. 将 Apache Software Foundation 许可证标头注释添加到所有新的 `.go` 文件（可以从项目中的现有文件复制）
4. 将您自己添加为您实质修改的 `.go` 文件的作者（不仅仅是外观更改）。
5. 添加文档
6. 进行一些单元测试也会有很大帮助。
7. 编写提交消息时，请遵循[该约定](https://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)，如果要解决现有问题，请在提交消息的末尾添加 “Fixes gh-XXXX”（其中 XXXX 是 issue 编号）。
8. 请确保代码覆盖率不会降低。
9. 将 PR 作为 Gitflow 工作流程的规则，并遵循 Pull Request 的规则。

## 版本命名约定

MOSN 的版本包含三位数，格式为 x.x.x，第一位是出于兼容性考虑； 第二个是新功能和增强功能； 最后一位是错误修复。

## 维护者代码审查策略

项目维护者审查代码时建议遵循以下策略：

1. 检查 PR 对应的 Issue
2. 检查解决方案的合理性
3. 检查 UT 和 Benchmark 的结果
4. 注意使代码结构发生变化的代码，全局变量的用法，特殊情况和并发的处理