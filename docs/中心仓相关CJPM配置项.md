# 中心仓相关配置项

cjpm.toml 中与制品包制作与发布有关的字段如下：

字段名 | 字段描述 | 是否必填
:-----|:----------|:--------
cjc-version | 仓颉 SDK 最低版本号 | 是
name | 模块名，与模块内 root 包名一致 | 是
organization | 组织名，为空则为无组织模块 | 否
description | 描述信息 | 仅打包时必填
version | 模块版本信息 | 是
output-type | 编译输出产物类型，取值为静态库/动态库/可执行 (static/dynamic/executable) | 是
authors | 作者 ID 列表 | 否
repository | 制品仓代码 url | 否
homepage | 制品主页 url | 否
documentation | 制品文档页 url | 否
tag | 制品标签 上限 5 个| 否
category | 官方提供的制品分类，取值范围详见 cjpm 文档相关章节 上限 5 个| 否
license | 协议列表 | 否
include | 指定打包范围 | 否
exclude | 指定打包排除范围 | 否
dependencies | 项目依赖，格式如：aoo = "1.0.0" 或 “org::boo” = { version = “2.0.0” } | 否
test-dependencies | 测试依赖，格式和 dependencies 相同 | 否
script-dependencies | 脚本依赖，格式和 dependencies 相同 | 否

如下是一个具体的 cjpm.toml 例子：
```toml
[package]
  cjc-version = "1.0.0"
  name = "demo"
  organization = "cangjie"
  description = "demo of cangjie central repository"
  version = "1.0.0"
  output-type = "executable"
  authors = ["Tom", "Joan"]
  repository = "cangjie-demo.git"
  homepage = "cangjie-demo.com"
  documentation = "cangjie-demo.com/docs"
  tag = ["cangjie", "demo"]
  category = ["Network", "UI"]
  license = ["Apache-2.0"]
  include = ["src"]
  exclude = ["*.txt"]
[dependencies]
  aoo = "1.0.0"
[test-dependencies]
  boo = "2.0.0"
[script-dependencies]
  "org::coo" = "3.0.0"
```

## 分类列表
该字段用于为源码模块指定制品分类，上限 5 个。

指定的制品分类需要在仓颉中心仓提供的制品分类范围内。仓颉中心仓制品分类列表如下：

分类名 | category
:----|:------
网络 | Network
数据库驱动 | Database Driver
数据封装传递 | Data Encapsulation and Transfer
数据解析 | Data Analysis
数据库框架 | Database Framework
对象存储 | Object Storage
分布式 | Distributed
任务调度 | Task Scheduling
安全类 | Security
工具类 | Utility
日志类 | Logging
算法类 | Algorithm
音视频 | Audio and Video
字符编码 | Character Encoding
图像处理 | Image Processing
开发者工具 | Developer Tools
动画类 | Animation
基础设施 | Infrastructure
地理信息 | Geographic Information
UI 类 | UI
科学计算 | Scientific Computing
编程框架 | Programming Framework
数据监控 | Data Monitoring
熔断降级 | Circuit Breaker and Downgrading
消息队列 | Message Queue

配置 `category` 字段时，配置的值大小写不敏感，但是在最后生成的元数据当中会被转化为上述的标准格式。


###　bundle 打包流程如下：

１. 模块检查
一个可以被成功打包的 cjpm 模块，需要满足以下条件：
 - 模块名、组织名满足规格要求：
    - 长度范围为 [3, 64]，且符合模块名和组织名规格；
    - 不能为任何仓颉语法中的关键字（大小写不敏感）；
    - 组织名不能为 default；
 - cjpm.toml 中包含模块说明 description；
 - 根目录下包含中文文档 README_zh.md 或英文文档 README.md；
 - 模块的依赖项均为中心仓形式。
 - 若模块不满足上述条件，则打包失败。

2. 编译检查：进行编译检查，确保模块能够编译通过；如果未配置 --skip-test，则会运行单元测试。编译和测试失败均会导致打包失败。

3. 代码静态检查：如果未配置 --skip-lint，则会调用 cjlint 进行代码静态检查，出现 error 级别的错误则会导致打包失败。

4. 打包：基于 include 和 exclude 字段，将当前模块打包成 tar.gz 格式的制品源码包。制品源码包位于编译产物目录中，文件名为 模块名-版本号.cjp。同时，生成制品包对应的元数据文件，也位于编译产物目录中，文件名为 meta-data.json。

继续以上文中待打包模块 demo 为例，假设其 cjpm.toml 中配置的版本号 version = "1.0.0"。由于未配置编译产物目录，因此默认编译产物目录为 target。执行 cjpm bundle 后，target 目录中会有如下内容：
```shell
target
├── demo-1.0.0.cjp  # 制品源码包
├── meta-data.json  # 制品元数据
└── 其他编译产物
```