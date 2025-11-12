# ManifestHub CLI

轻量级的命令行工具, 用于提取 Steam 创意工坊 / 游戏 相关的密钥, 基于 Golang 编写。

## 主要特性

- 使用 Go 编写, 单文件二进制, 便于分发与部署
- 支持对大部分 Steam 游戏的提取 (包括较新的游戏)
- 支持提取创意工坊密钥、游戏密钥以及无 Depot 的 DLC 信息
- 内置基本的配置文件(`config.ini`)

## 仓库结构

- `main.go`: 程序入口
- `config.go` / `config.ini`: 配置与默认选项
- `download.go`: 下载/解析相关实现
- `process.go`: 处理与转换逻辑
- `user.go`: 用户与凭据相关逻辑
- `defs.go`: 类型与常量定义
- `.gitignore`: 在 Git 中忽略文件和目录

## 配置

项目提供 `config.ini`(示例)作为默认配置文件, 主要配置项包括: 

- 下载源地址(默认列表)

## 使用方法

1. 双击打开程序
2. 输入游戏名称/AppID/Steam链接/SteamDB链接
3. 选择对应的游戏 (如果是直接输入AppID则没有此步骤)
4. 自动下载到指定文件中 (默认为当前程序所在的文件夹)
5. 对 .lua 文件进行对应的处理 (如: 拖入 SteamTools 悬浮窗口)

## 开发环境需求

- Go 1.18 或更高版本(用于本地构建)
- 网络访问权限(下载 manifest/资源)

## 打包与发布

下面是常见打包的方法:

```shell
# 正常打包 (无参数传入)
go build .
```

```shell
# 最小化打包 (生成环境推荐)
go build -ldflags="-s -w -buildid=" -trimpath .
```

## 常见问题与注意事项

- 请确保网络与目标源可达；某些清单/资源可能受访问限制。
- 本工具只提供技术手段, 请勿用于侵犯他人合法权益的用途。

## 贡献

欢迎贡献代码、文档与 issue。提交 PR 前请: 

1. Fork 本仓库并在分支上实现修改
2. 保持代码风格一致并添加必要注释
3. 提交 PR 并在描述中说明变更内容与测试方式

## 参考与资源

- 官方/相关项目仓库(仅供参考): 
	- SteamAutoCracks/ManifestHub: https://github.com/SteamAutoCracks/ManifestHub
	- Walftech: https://walftech.com/

---
本项目遵循 MIT License 许可证协议