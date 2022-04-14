# MOSN 贡献者指南

MOSN 基于 Apache 2.0 许可发布，遵循标准的 Github 开发流程，使用 Github Issue 跟踪问题并将 Pull Request 合并到 master 分支中。本文档可以帮助你了解如何参与贡献。

在提交 Pull Request 之前您需要先签署[贡献者许可协议（CLA）](http://cla.sofastack.tech/)。虽然签署 CLA 不会给您授予主存储库的提交权利，但这意味着我们可以接受您的贡献，签署 CLA 后您将成为本项目的荣誉作者。对于活跃的贡献者甚至可以加入我们的核心团队，并具有合并 Pull Request 的权限。

## 建议

为了能更高效的推进 PR 的合并，我们有一些推荐的做法，希望能有所帮助。

1. 创建分支
   推荐使用新分支来开发，master 分支推荐保持跟 MOSN 上游保持一致
2. PR 需要说明意图
   如果已经有对应讨论的 issue，可以引用 issue
   如果没有 issue，需要描述清楚 PR 的意图，比如 bug 的情况。
   如果改动比较大，最好可以比较详细的改动说明介绍。
3. 提交新的 commit 来处理 review 意见
   当 PR 收到 review 意见后，有新的改动，推荐放到新的 commit，不要追加到原来的 commit，这样方便 reviewer 查看新的变更
4. 尽量提交小 PR
   不相关的改动，尽量放到不同的 PR，这样方便快速 review 合并
5. 尽量减少 force push
   因为 force push 之后，review 意见就对不上原始的代码记录了，这样不利于其他人了解 review 的过程。除非是因为需要 rebase 处理跟 master 的冲突，这种 rebase 后就只能 force push 了。

## commit log

commit log 推荐以一个单词开头，比如下面的这些：

1. feature: 实现了一个新的功能/特性
2. change: 没有向后兼容的变更
3. refactor: 代码重构
4. bugfix: bug 修复
5. optimize: 性能优化相关的变更
6. doc: 文档变更，包括注释
7. tests: 测试用例相关的变更
8. style: 代码风格相关的调整
9. sample: 示例相关的变更
10. chore: 其他不涉及核心逻辑的小改动

开头单词之后，是简要介绍一下改动的内容，比如新增了什么功能，如果是 bugfix 的话，还需要说明 bug 复现的条件，以及危害。
比如：

```
bugfix: got the wrong CACert filename when converting the listen filter from istio LDS, mosn may not listen success then.
```

如果比较复杂，一句话写不清楚的话，也可以写多行，commit log 不用怕太长。

如果英文不容易写清楚，在 PR comment 里用中文描述清楚也可以的。

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
7. 请确保代码覆盖率不会降低。
8. 将 PR 作为 Gitflow 工作流程的规则，并遵循 Pull Request 的规则。

## 版本命名约定

MOSN 的版本包含三位数，格式为 x.x.x，第一位是出于兼容性考虑； 第二个是新功能和增强功能； 最后一位是错误修复。

## 维护者代码审查策略

项目维护者审查代码时建议遵循以下策略：

1. 检查 PR 对应的 Issue
2. 检查解决方案的合理性
3. 检查 UT 和 Benchmark 的结果
4. 注意使代码结构发生变化的代码，全局变量的用法，特殊情况和并发的处理

合并的时候：

1. 检查 commit log 是否标准，可以适当修改
2. github 自动生成的 squash log 通常都没用，最好重新编辑，调整一下