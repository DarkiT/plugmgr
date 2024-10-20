# Go 语言的插件管理器

一个用于 Go 应用程序的高性能、可扩展的插件管理库。

## 功能特点

- 动态加载和卸载插件
- 版本控制和依赖管理
- 热重载和延迟加载
- 生命周期事件系统
- 增强的沙盒功能以提高安全性
- 插件性能的指标收集
- 灵活的配置管理，支持二进制数据
- 插件签名验证
- 高并发支持
- 优化的内存管理

## 项目结构

```
.
├── config.go         // 配置管理
├── discovery.go      // 插件发现和验证
├── errors.go         // 错误定义
├── event.go          // 事件系统
├── manager.go        // 插件管理器核心
├── plugin.go         // 插件接口和相关结构
├── sandbox.go        // 沙箱接口
├── sandbox_other.go  // 非 Windows 平台的沙箱实现
└── sandbox_windows.go // Windows 平台的沙箱实现（不支持）
```

## 入门指南

使用 Go 包管理器获取最新版本的库：

```bash
go get github.com/darkit/plugins
```

将包导入你的源码中：

```go
import "github.com/darkit/plugins"
```

## 使用方法

### 初始化新的插件管理器

```go
manager, err := plugins.NewManager("plugins", "config.msgpack", "public_key.pem")
if err != nil {
    log.Fatalf("初始化插件管理器失败: %v", err)
}
```

### 加载插件

```go
err = manager.LoadPluginWithData("./plugins/myplugin.so", initialConfig)
if err != nil {
    log.Printf("加载插件失败: %v", err)
}
```

### 执行插件

```go
result, err := manager.ExecutePlugin("MyPlugin", data)
if err != nil {
    log.Printf("执行插件失败: %v", err)
} else {
    log.Printf("插件执行结果: %v", result)
}
```

### 卸载插件

```go
err = manager.UnloadPlugin("MyPlugin")
if err != nil {
    log.Printf("卸载插件失败: %v", err)
}
```

### 热重载插件

```go
err = manager.HotReload("MyPlugin", "./plugins/myplugin_v2.so")
if err != nil {
    log.Printf("热重载插件失败: %v", err)
}
```

### 启用自动插件发现

```go
err = manager.DiscoverPlugins("./plugins")
if err != nil {
    log.Printf("插件发现失败: %v", err)
}
```

### 订阅插件事件

```go
manager.SubscribeToEvent("PluginLoaded", func(e plugins.Event) {
    if loadedEvent, ok := e.(plugins.PluginLoadedEvent); ok {
        log.Printf("插件已加载: %s", loadedEvent.PluginName)
    }
})
```

### 更新插件配置

```go
updatedConfig, err := manager.ManagePluginConfig("MyPlugin", newConfig)
if err != nil {
    log.Printf("更新插件配置失败: %v", err)
}
```

## 创建插件

插件必须实现 `Plugin` 接口。以下是一个简单的插件实现示例：

```go
type MyPlugin struct {
    Name    string
    Version string
    Config  interface{}
}

func (p *MyPlugin) Metadata() plugins.PluginMetadata {
    return plugins.PluginMetadata{
        Name:         p.Name,
        Version:      p.Version,
        Dependencies: map[string]string{},
    }
}

func (p *MyPlugin) PreLoad(config []byte) error {
    return plugins.Deserializer(config, &p.Config)
}

func (p *MyPlugin) Init() error {
    fmt.Println("MyPlugin: Init called")
    return nil
}

// 实现其他接口方法...

var Plugin MyPlugin = MyPlugin{
    Name:    "MyPlugin",
    Version: "1.0.0",
}
```

## 编译插件

使用以下命令编译插件：

```bash
go build -buildmode=plugin -o myplugin.so myplugin.go
```

## 配置

插件管理器现在使用 MessagePack 格式的配置文件来存储插件配置和状态，提供更高的性能和更小的文件大小。

## 插件签名

为了增加安全性，可以对插件进行签名验证：

```go
err = manager.VerifyPluginSignature(pluginPath, publicKeyPath)
if err != nil {
    log.Printf("插件签名验证失败: %v", err)
}
```

## 性能优化

- 使用 sync.Map 替代普通 map，提高并发安全性
- 使用 MessagePack 替代 GOB 进行序列化，提高性能
- 实现了插件缓存，减少重复加载
- 使用原子操作更新统计信息，确保线程安全
- 优化了错误处理，提供更详细的错误信息

## 注意事项

1. 该插件系统目前不支持 Windows 平台。
2. 请确保插件实现了所有必要的接口方法。
3. 在生产环境中使用时，建议启用插件签名验证以增强安全性。
4. 插件的依赖关系应该谨慎管理，避免出现循环依赖。
5. 虽然提供了沙箱功能，但仍应该仔细审查插件代码以确保安全性。

## 许可证

MIT License