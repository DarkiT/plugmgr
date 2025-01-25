package plugmgr

import (
	"plugin"
	"sync"
	"time"
)

var (
	pluginCache   sync.Map
	pluginCacheMu sync.Mutex
)

type PluginMetadata struct {
	Name         string
	Version      string
	Dependencies map[string]string
	GoVersion    string
	Signature    []byte
	Config       any
}

// Plugin 定义了插件必须实现的接口
type Plugin interface {
	// Metadata 返回插件的元数据信息
	// 包含插件名称、版本、依赖关系等基本信息
	Metadata() PluginMetadata

	// Init 插件初始化
	// 在插件加载后执行，用于初始化插件的资源和状态
	Init() error

	// PostLoad 加载后处理
	// 在插件初始化完成后执行，用于处理依赖项和建立连接
	PostLoad() error

	// PreUnload 卸载前处理
	// 在插件卸载前执行，用于清理资源和保存状态
	PreUnload() error

	// Shutdown 关闭插件
	// 在插件完全卸载前执行，用于释放所有资源
	Shutdown() error

	// PreLoad 加载前处理
	// 在插件初始化前执行，用于加载配置和验证环境
	// 参数 config: 插件的配置数据
	PreLoad(config []byte) error

	// ConfigUpdated 管理插件配置
	// 用于更新和维护插件的配置信息
	// 参数 config: 新的配置数据
	// 返回: 处理后的配置数据和可能的错误
	ConfigUpdated(config []byte) ([]byte, error)

	// Execute 执行插件功能
	// 插件的主要业务逻辑实现
	// 参数 data: 输入数据
	// 返回: 处理结果和可能的错误
	Execute(data any) (any, error)
}

type PluginStats struct {
	ExecutionCount     int64
	LastExecutionTime  time.Duration
	TotalExecutionTime time.Duration
}

// LoadPlugin 加载插件
func LoadPlugin(path string) (Plugin, error) {
	if cachedPlugin, ok := pluginCache.Load(path); ok {
		return cachedPlugin.(Plugin), nil
	}

	pluginCacheMu.Lock()
	defer pluginCacheMu.Unlock()

	// 双重检查
	if cachedPlugin, ok := pluginCache.Load(path); ok {
		return cachedPlugin.(Plugin), nil
	}

	p, err := plugin.Open(path)
	if err != nil {
		return nil, wrapf(err, "打开插件失败: %s", path)
	}

	symPlugin, err := p.Lookup(PluginSymbol)
	if err != nil {
		return nil, wrapf(err, "查找插件符号失败: %s", path)
	}

	pluginInfo, ok := symPlugin.(Plugin)
	if !ok {
		return nil, wrapf(ErrInvalidPluginInterface, "插件接口无效: %s", path)
	}

	pluginCache.Store(path, pluginInfo)

	return pluginInfo, nil
}

const PluginSymbol = "Plugin"
