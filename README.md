# Go 语言的插件管理器

一个用于 Go 应用程序的强大且灵活的插件管理库。

## 功能特点

- 动态加载和卸载
- 版本控制和依赖管理
- 热重载和延迟加载
- 生命周期事件系统
- 增强的沙盒功能以提高安全性
- 插件性能的指标收集
- 灵活的配置管理，支持二进制数据
- 插件签名验证

## 入门指南

使用 Go 包管理器获取最新版本的 `plugin-manager` 库（推荐）：

```bash
go get github.com/darkit/plugins
```

或者使用 Git 命令行将仓库克隆到本地计算机：

```sh
git clone https://github.com/darkit/plugins.git
cd plugin-manager
```

获取库后，将包导入你的源码中：

```go
import "github.com/darkit/plugins"
```

## 使用方法

请访问 [examples]() 目录查看完整功能实现示例。

### 初始化新的插件管理器

在应用程序启动时创建一个新的插件管理器实例。

```go
manager, err := pm.NewManager("plugins", "config.gob", "public_key.pem")
```

**参数:**

- `pluginDir`（字符串）：存储插件的目录。
- `configPath`（字符串）：用于管理插件配置的 GOB 文件路径。
- `publicKeyPath`（字符串,可选）：用于验证插件签名的公钥文件路径。

**返回值:**

- `*Manager`：指向新创建的 Manager 实例的指针。
- `error`：初始化过程中遇到的任何错误。

### 加载插件

从指定路径加载插件到内存中，使其可供执行。

```go
err = manager.LoadPluginWithData("./plugins/myplugin.so", initialConfig)
```

**参数:**

- `path`（字符串）：插件文件的路径（.so 扩展名）。
- `data`（...interface{}）：可选的初始配置数据。

**返回值:**

- `error`：加载过程中遇到的任何错误。

### 执行插件

运行已加载插件的 `Execute()` 方法。

```go
err = manager.ExecutePlugin("MyPlugin", data...)
```

**参数:**

- `name`（字符串）：要执行的插件名称。
- `data`（...interface{}）：传递给插件 Execute 方法的数据。

**返回值:**

- `error`：插件执行过程中遇到的任何错误。

### 卸载插件

当插件不再需要时，从内存中安全地移除它。

```go
err = manager.UnloadPlugin("MyPlugin")
```

**参数:**

- `name`（字符串）：要卸载的插件名称。

**返回值:**

- `error`：卸载过程中遇到的任何错误。

### 热重载插件

在应用程序运行时（无需停止应用程序）更新已加载的插件到新版本。

```go
err = manager.HotReload("MyPlugin", "./plugins/myplugin_v2.so")
```

**参数:**

- `name`（字符串）：要热重载的插件名称。
- `path`（字符串）：插件新版本的路径。

**返回值:**

- `error`：热重载过程中遇到的任何错误。

### 启用自动插件发现

自动发现并加载指定目录中的所有插件。

```go
err = manager.DiscoverPlugins("./plugins")
```

**参数:**

- `dir`（字符串）：搜索插件的目录。

**返回值:**

- `error`：发现过程中的任何错误。

### 订阅插件事件

订阅特定插件事件，在事件发生时执行提供的函数。

```go
manager.SubscribeToEvent("PluginLoaded", func(e pm.Event) {
    fmt.Printf("插件加载: %s\n", e.(pm.PluginLoadedEvent).PluginName)
})
```

**参数:**

- `eventName`（字符串）：要订阅的事件名称（如 "PluginLoaded"）。
- `handler`（func(Event)）：事件发生时执行的函数。

**返回值:** 无

### 更新插件配置

在运行时更新插件的配置。

```go
err = manager.UpdatePluginConfig("MyPlugin", newConfig)
```

**参数:**

- `name`（字符串）：要更新配置的插件名称。
- `config`（interface{}）：新的配置数据。

**返回值:**

- `error`：更新配置过程中遇到的任何错误。

## 创建插件

插件必须实现 `Plugin` 接口。

### Plugin 接口

```go
type Plugin interface {
    Metadata() PluginMetadata
    PreLoad(config interface{}) error
    Init() error
    PostLoad() error
    Execute(data ...interface{}) error
    PreUnload() error
    Shutdown() error
    UpdateConfig(config interface{}) error
}
```

### 插件结构体示例

```go
type MyPlugin struct {
    Name    string
    Version string
    Config  interface{}
}
```

### 实现插件接口

```go
func (p *MyPlugin) Metadata() pm.PluginMetadata {
    return pm.PluginMetadata{
        Name:         p.Name,
        Version:      p.Version,
        Dependencies: map[string]string{},
    }
}

func (p *MyPlugin) PreLoad(config interface{}) error {
    p.Config = config
    fmt.Println("MyPlugin: PreLoad called")
    return nil
}

func (p *MyPlugin) Init() error {
    fmt.Println("MyPlugin: Init called")
    return nil
}

func (p *MyPlugin) PostLoad() error {
    fmt.Println("MyPlugin: PostLoad called")
    return nil
}

func (p *MyPlugin) Execute(data ...interface{}) error {
    fmt.Println("MyPlugin: Execute called with data:", data)
    return nil
}

func (p *MyPlugin) PreUnload() error {
    fmt.Println("MyPlugin: PreUnload called")
    return nil
}

func (p *MyPlugin) Shutdown() error {
    fmt.Println("MyPlugin: Shutdown called")
    return nil
}

func (p *MyPlugin) UpdateConfig(config interface{}) error {
    p.Config = config
    fmt.Println("MyPlugin: UpdateConfig called")
    return nil
}
```

### 导出插件

```go
var Plugin MyPlugin = MyPlugin{
    Name:    "MyPlugin",
    Version: "1.0.0",
}

// 导出插件
var ExportedPlugin struct{}

func (pe *ExportedPlugin) NewPlugin() interface{} {
    return &Plugin
}
```

## 编译插件

通过设置 `-buildmode` 标志为 `plugin` 使用标准 Go 编译器工具链编译插件：

```bash
go build -buildmode=plugin -o myplugin.so myplugin.go
```

## 配置

插件管理器使用 GOB 格式的配置文件来存储插件配置和状态。这允许存储复杂的数据结构，包括二进制数据。

## 插件签名

为了增加安全性，可以对插件进行签名验证。使用 `VerifyPluginSignature` 方法来验证插件的签名。

```go
err = manager.VerifyPluginSignature(pluginPath, publicKeyPath)
```

## 较原插件升级了以下内容：
```
1. 更新了 `NewManager` 函数的参数，现在包括可选的公钥路径。
2. 添加了 `LoadPluginWithData` 方法的说明，支持传递初始配置数据。
3. 更新了 `ExecutePlugin` 方法，现在支持传递数据给插件的 Execute 方法。
4. 添加了 `UpdatePluginConfig` 方法的说明。
5. 更新了 Plugin 接口的定义，包括新的 `UpdateConfig` 方法和修改后的 `PreLoad` 和 `Execute` 方法签名。
6. 更新了插件实现的示例代码，反映了新的接口要求。
7. 添加了关于配置文件格式变更（从 JSON 到 GOB）的说明。
8. 添加了关于插件签名验证的部分。
```

## 许可证

MIT License
