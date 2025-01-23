# Go 语言插件管理器

一个用于Go应用程序的插件管理库，支持动态加载、热重载、依赖管理及 Web 框架适配。

## 功能特点

- 动态加载和卸载插件
- 版本控制和依赖管理
- 热重载支持
- 生命周期事件系统
- 增强的沙箱功能以提高安全性
- 插件性能的指标收集
- 灵活的配置管理
- 插件签名验证
- 高并发支持
- 优化的内存管理
- 支持多种 Web 框架适配器 (Gin, 标准 net/http)

---

## 安装与快速开始

### 安装

使用 Go 包管理器获取最新版本：

```bash
go get github.com/darkit/plugmgr
```

### 基础使用

```go
package main

import (
    "log"
    pm "github.com/darkit/plugmgr"
)

func main() {
    // 初始化插件管理器
    manager, err := pm.NewManager(
        "./plugins",        // 插件目录
        "./config.db",    // 配置文件路径
        "./public_key.pem", // 可选的公钥路径
    )
    if err != nil {
        log.Fatal(err)
    }

    // 加载插件
    err = manager.LoadPlugin("./plugins/demo.so")
    if err != nil {
        log.Printf("加载插件失败: %v", err)
    }

    // 执行插件
    result, err := manager.ExecutePlugin("demo", data)
    if err != nil {
        log.Printf("执行插件失败: %v", err)
    }
}
```

### 初始化插件管理器

```go
manager, err := pm.NewManager("./plugins", "config.db", "public_key.pem")
```

**参数:**

- `pluginDir`（字符串）：存储插件的目录。
- `configPath`（字符串）：用于管理插件启用/禁用的 JSON 配置文件。
- `publicKeyPath`（字符串，可选）：用于验证插件签名的公钥文件路径。

### 加载、执行和卸载插件

```go
err = manager.LoadPlugin("./plugins/myplugin.so")
err = manager.ExecutePlugin("MyPlugin")
err = manager.UnloadPlugin("MyPlugin")
err = manager.HotReload("MyPlugin", "./plugins/myplugin_v2.so")
```

---

## Web 框架集成

### Gin 框架

```go
package main

import (
    "log"
    pm "github.com/darkit/plugmgr"
    "github.com/darkit/plugmgr/adapter/gin"
    "github.com/gin-gonic/gin"
)

func main() {
    manager, _ := pm.NewManager("./plugins", "config.db")
    
    r := gin.Default()
    
    // 创建并注册 Gin 适配器
    handler := gin.NewGinHandler(manager)
    handler.RegisterRoutes(r)
    
    r.Run(":8080")
}
```

### 标准 HTTP

```go
package main

import (
    "log"
    "net/http"
    pm "github.com/darkit/plugmgr"
    "github.com/darkit/plugmgr/adapter/http"
)

func main() {
    manager, _ := pm.NewManager("./plugins", "config.db")
    
    // 创建并注册 HTTP 适配器
    handler := http.NewHTTPHandler(manager)
    
    log.Fatal(http.ListenAndServe(":8080", handler.RegisterRoutes()))
}
```

---

## 插件开发指南

插件需实现 `Plugin` 接口：

```go
type Plugin interface {
    // Metadata 返回插件的元数据信息
    // 包括：插件名称、版本、作者、描述、依赖等
    Metadata() PluginMetadata

    // Init 插件初始化
    // 在插件首次加载时调用，用于初始化资源
    Init() error

    // PostLoad 加载后处理
    // 在插件加载完成后调用，可以进行一些额外的设置
    PostLoad() error

    // PreUnload 卸载前处理
    // 在插件卸载前调用，用于清理资源
    PreUnload() error

    // Shutdown 关闭处理
    // 在插件管理器关闭时调用，用于释放资源
    Shutdown() error

    // PreLoad 加载前配置处理
    // 在插件加载前调用，用于处理初始配置
    // config: 包含插件的初始配置数据
    PreLoad(config []byte) error

    // ManageConfig 配置管理
    // 处理插件的配置更新
    // config: 新的配置数据
    // 返回: 处理后的配置数据和可能的错误
    ManageConfig(config []byte) ([]byte, error)

    // Execute 执行插件功能
    // data: 输入参数
    // 返回: 执行结果和可能的错误
    Execute(data any) (any, error)
}
```


每个方法在插件的生命周期中扮演不同的角色：

1. **元数据管理**
   - `Metadata()`: 提供插件的基本信息，用于插件管理和版本控制

2. **生命周期钩子**
   - `Init()`: 插件的初始化阶段
   - `PostLoad()`: 插件加载完成后的处理
   - `PreUnload()`: 插件卸载前的清理工作
   - `Shutdown()`: 插件关闭时的资源释放

3. **配置管理**
   - `PreLoad()`: 初始配置的预处理
   - `ManageConfig()`: 运行时配置的更新和管理

4. **功能执行**
   - `Execute()`: 插件核心功能的执行入口

**插件生命周期事件**

```go
manager.SubscribeToEvent("PluginLoaded", func(e pm.Event) {
    fmt.Printf("插件加载: %s\n", e.(pm.PluginLoadedEvent).PluginName)
})
manager.SubscribeToEvent("PluginUnloaded", func(e pm.Event) {
    fmt.Printf("插件卸载: %s\n", e.(pm.PluginLoadedEvent).PluginName)
})
manager.SubscribeToEvent("PluginHotReloaded", func(e pm.Event) {
    fmt.Printf("插件热加载: %s\n", e.(pm.PluginLoadedEvent).PluginName)
})
```

### 插件示例

```go
type MyPlugin struct{}

func (p *MyPlugin) Metadata() pm.PluginMetadata {
    return pm.PluginMetadata{
        Name:         "MyPlugin",
        Version:      "1.0.0",
        Dependencies: map[string]string{},
    }
}

func (p *MyPlugin) Init() error {
    fmt.Println("插件初始化")
    return nil
}

func (p *MyPlugin) PostLoad() error {
    fmt.Println("插件加载后处理")
    return nil
}

func (p *MyPlugin) PreUnload() error {
    fmt.Println("插件卸载前处理")
    return nil
}

func (p *MyPlugin) Shutdown() error {
    fmt.Println("插件关闭")
    return nil
}

func (p *MyPlugin) PreLoad(config []byte) error {
    fmt.Println("插件预加载配置")
    return nil
}

func (p *MyPlugin) ManageConfig(config []byte) ([]byte, error) {
    fmt.Println("管理插件配置")
    return config, nil
}

func (p *MyPlugin) Execute(data any) (any, error) {
    fmt.Println("插件执行")
    return nil, nil
}
```

### 插件变量

插件管理器通过 `Plugin` 变量发现你的插件，必须定义如下：

```go
var Plugin *MyPlugin
```

### 编译插件

```bash
go build -buildmode=plugin -o myplugin.so myplugin.go
```

---

## 远程插件库与自动更新

```go
repo, err := manager.SetupRemoteRepository("user@example.com:/path/to/repo")
```

### 安装 Redbean 作为插件服务器

```go
err := manager.InstallRedbean("1.5.3", "./redbean", "user@example.com:/path/to/repo")
```

**优势：**

- **轻量级**：Redbean 体积小，占用资源少。
- **高效性**：提供快速插件分发能力。
- **安全性**：内置插件签名验证。

---
## REST API 支持

| 方法     | 端点                       | 说明             |
| -------- | ------------------------ | -------------- |
| `GET`    | `/plugins`               | 获取已加载的插件列表 |
| `POST`   | `/plugins/:name/load`    | 加载插件        |
| `POST`   | `/plugins/:name/unload`  | 卸载插件        |
| `POST`   | `/plugins/:name/execute` | 执行插件        |
| `GET`    | `/plugins/:name/config`  | 获取插件配置     |
| `PUT`    | `/plugins/:name/config`  | 更新插件配置     |
| `POST`   | `/plugins/:name/enable`  | 启用插件        |
| `POST`   | `/plugins/:name/disable` | 禁用插件        |

---

## 安全性

- **插件签名验证**：确保插件来源可信。
- **沙箱隔离**：防止插件影响宿主应用。
- **权限控制**：限制插件访问的资源。

---

## 项目结构

```
.
├── adapter/                   // Web 框架适配器
│   ├── gin/                   // Gin 框架适配器
│   ├── http/                  // 标准 http 适配器
│   └── adapter.go             // 适配器接口
├── docs/                      // 文档
│   ├── PluginSignature.md     // 插件签名指南
│   └── Redbean.md             // Redbean 配置说明
├── examples/                  // 示例代码
│   ├── gin/                   // Gin 框架示例
│   └── plugins/               // 插件示例
├── config.go                  // 配置管理
├── discovery.go               // 插件发现和验证
├── errors.go                  // 错误定义
├── event.go                   // 事件系统
├── logger.go                  // 日志接口
├── manager.go                 // 插件管理器核心
├── plugin.go                  // 插件接口和相关结构
├── sandbox.go                 // 沙箱接口
├── sandbox_other.go           // 非 Windows 平台的沙箱实现
└── sandbox_windows.go         // Windows 平台的沙箱实现
```

---

## 许可证 & 贡献

本项目采用 MIT License 许可证，欢迎提交 Issue 和 Pull Request！

