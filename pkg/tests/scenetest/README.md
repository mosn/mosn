## 说明

+ 测试用例在scene开头的文件里
+ 通用的定义在util开头的文件里
+ 目前想要关闭mesh只能通过关闭进程，因此必须指定某一个测试用例进行测试

···
go test -v -run TestXXX
···

## Introduction

+ The test cases are in the files that name start with scene
+ You can find the common definition  in the files that start with util
+ You must specify a test case to run, because the mesh can only close by terminate the process

···
go test -v -run TestXXX
···
