# Go 语言插件管理器

一个用于Go应用程序的插件管理库，支持动态加载、热重载、依赖管理及 Web 框架适配。旨在提供一个轻量、安全、高性能的插件系统，使开发者能够轻松地构建可扩展的应用程序。通过解耦核心应用程序和插件，我们实现了更灵活的架构和更好的可维护性。

## 功能特点

- 动态加载和卸载插件
- 版本控制和依赖管理
- 热重载支持
- 生命周期事件系统
- 沙箱功能以提高安全性
- 插件性能的指标收集
- 灵活的配置管理
- 插件签名验证
- 支持多种 Web 框架适配器

## 主要优势

1. **模块化设计**：将应用程序功能解耦，提高系统的可扩展性
2. **安全隔离**：通过沙箱机制确保插件运行的安全性
3. **动态管理**：支持运行时加载、卸载和热重载插件
4. **跨平台兼容**：支持主流操作系统和 Web 框架
5. **性能优化**：最小化插件加载和执行的性能开销

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
        "./config.db",      // 配置文件路径
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
- `configPath`（字符串）：用于管理插件启用/禁用的配置文件。
- `publicKeyPath`（字符串，可选）：用于验证插件签名的公钥文件路径。

### 加载、执行和卸载插件

```go
err = manager.LoadPlugin("./plugins/myplugin.so")
err = manager.ExecutePlugin("MyPlugin")
err = manager.UnloadPlugin("MyPlugin")
err = manager.HotReload("MyPlugin", "./plugins/myplugin_v2.so")
```

## Web 框架集成

### 通用适配器接口

插件管理器提供了一个通用的适配器接口，可以轻松集成到不同的 Web 框架中：

```go
type Handler[T any] interface {
    // 基础插件管理
    ListPlugins() T
    LoadPlugin() T
    UnloadPlugin() T
    EnablePlugin() T
    DisablePlugin() T
    PreloadPlugin() T
    HotReloadPlugin() T

    // 插件配置
    GetPluginConfig() T
    PluginConfigUpdated() T

    // 插件执行
    ExecutePlugin() T

    // 插件权限
    GetPluginPermission() T
    SetPluginPermission() T
    RemovePluginPermission() T

    // 插件市场
    ListMarketPlugins() T
    InstallPlugin() T
    RollbackPlugin() T

    // 插件统计
    GetPluginStats() T
}
```

### 标准 HTTP 集成

```go
// 创建 HTTP 处理器
Http := adapter.NewPluginHandler(manager, func(h http.HandlerFunc) http.HandlerFunc {
    return h
})

// 获取处理器接口
HttpHandlers := Http.GetHandlers()

// 创建路由
mux := http.NewServeMux()

// 设置路由
mux.HandleFunc("/plugins", HttpHandlers.ListPlugins())
mux.HandleFunc("/plugins/load/", HttpHandlers.LoadPlugin())
// ... 其他路由

log.Fatal(http.ListenAndServe(":8080", mux))
```

### Gin 框架集成

```go
// 创建 Gin 处理器
Gin := adapter.NewPluginHandler(manager, func(h http.HandlerFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        h(c.Writer, c.Request)
    }
})

// 获取处理器接口
GinHandlers := Gin.GetHandlers()

// 创建 Gin 路由
r := gin.Default()

// 设置路由
r.GET("/plugins", GinHandlers.ListPlugins())
r.POST("/plugins/load/:name", GinHandlers.LoadPlugin())
// ... 其他路由

log.Fatal(r.Run(":8080"))
```

### API 端点说明

| 方法   | 路径                        | 说明               |
|--------|---------------------------|-------------------|
| GET    | /plugins                  | 获取插件列表         |
| POST   | /plugins/load/:name       | 加载插件           |
| POST   | /plugins/unload/:name     | 卸载插件           |
| POST   | /plugins/enable/:name     | 启用插件           |
| POST   | /plugins/disable/:name    | 禁用插件           |
| POST   | /plugins/preload/:name    | 预加载插件         |
| POST   | /plugins/hotreload/:name  | 热重载插件         |
| GET    | /plugins/config/:name     | 获取插件配置        |
| PUT    | /plugins/config/:name     | 更新插件配置        |
| GET    | /plugins/permission/:name | 获取插件权限        |
| PUT    | /plugins/permission/:name | 设置插件权限        |
| DELETE | /plugins/permission/:name | 移除插件权限        |
| GET    | /plugins/stats/:name      | 获取插件统计信息     |
| GET    | /market                  | 获取插件市场列表     |
| POST   | /market/install/:name    | 安装插件           |
| POST   | /market/rollback/:name   | 回滚插件版本        |

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

    // ConfigUpdated 配置管理
    // 处理插件的配置更新
    // config: 新的配置数据
    // 返回: 处理后的配置数据和可能的错误
    ConfigUpdated(config []byte) ([]byte, error)

    // Execute 执行插件功能
    // data: 输入参数
    // 返回: 执行结果和可能的错误
    Execute(data any) (any, error)
}
```

## 插件生命周期与事件系统

### 事件系统概述

插件管理器提供了完整的事件系统，用于监控和响应插件的各种状态变化。事件系统是插件生命周期管理的核心组件，允许开发者在插件的不同阶段进行精细化的监控和处理。

### 内置事件类型

| 事件名称 | 触发时机 | 事件数据 |
|---------|---------|---------|
| `PluginLoaded` | 插件加载完成时 | 插件名称 |
| `PluginInitialized` | 插件初始化完成时 | 插件名称 |
| `PluginExecuted` | 插件执行完成时 | 插件名称、执行结果 |
| `PluginConfigUpdated` | 插件配置更新时 | 插件名称、新配置 |
| `PluginPreUnload` | 插件卸载前 | 插件名称 |
| `PluginUnloaded` | 插件卸载完成时 | 插件名称 |
| `PluginExecutionError` | 插件执行出错时 | 插件名称、错误信息 |
| `PluginHotReloaded` | 插件热重载完成时 | 插件名称 |

### 事件订阅

#### 基本订阅

```go
// 创建事件总线（可选配置超时时间）
eventBus := pm.NewEventBus(
    pm.WithTimeout(3 * time.Second), // 设置事件处理超时时间
)

// 订阅插件加载事件
eventBus.Subscribe("PluginLoaded", func(e pm.Event) {
    fmt.Printf("插件已加载: %s\n", e.Data.Name)
})

// 订阅插件执行错误事件
eventBus.Subscribe("PluginExecutionError", func(e pm.Event) {
    fmt.Printf("插件执行错误: %s, 错误: %v\n", 
        e.Data.Name, 
        e.Data.Error,
    )
})
```

#### 事件发布方式

支持同步和异步两种事件发布方式：

```go
// 同步发布（阻塞等待所有处理器执行完成）
err := eventBus.Publish(pm.Event{
    EventName: "PluginLoaded",
    Data: pm.EventData{
        Name: "demo",
    },
})

// 异步发布（带超时控制）
err := eventBus.PublishAsync(pm.Event{
    EventName: "PluginExecuted",
    Data: pm.EventData{
        Name: "demo",
        Data: result,
    },
})
```

### 事件管理功能

```go
// 取消订阅事件
handler := func(e pm.Event) { ... }
eventBus.Unsubscribe("PluginLoaded", handler)

// 检查事件是否有订阅者
if eventBus.HasSubscribers("PluginLoaded") {
    // ...
}

// 获取事件订阅者数量
count := eventBus.SubscribersCount("PluginLoaded")

// 关闭事件总线
err := eventBus.Close()
```

### 事件处理特性

- **超时控制**：可配置事件处理超时时间，防止处理器阻塞
- **并发安全**：支持多个 goroutine 同时订阅和发布事件
- **优雅关闭**：支持事件总线的安全关闭
- **灵活订阅**：支持同一事件多个处理器
- **处理器隔离**：单个处理器的错误不影响其他处理器

## 插件示例

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

func (p *MyPlugin) ConfigUpdated(config []byte) ([]byte, error) {
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

## 安全性

- **插件签名验证**：确保插件来源可信。
- **沙箱隔离**：防止插件影响宿主应用。
- **权限控制**：限制插件访问的资源。

## 项目结构

```
.
├── adapter/                   // Web 框架适配器
│   └── adapter.go             // 适配器接口
├── docs/                      // 文档
│   ├── PluginSignature.md     // 插件签名指南
│   └── Redbean.md             // Redbean 配置说明
├── examples/                  // 示例代码
│   ├── http/                  // Http 框架示例
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

## 许可证 & 贡献

本项目采用 MIT License 许可证，欢迎提交 Issue 和 Pull Request！