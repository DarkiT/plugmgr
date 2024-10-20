package pluginmanager

import (
	"plugin"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type PluginMetadata struct {
	Name         string
	Version      string
	Dependencies map[string]string
	GoVersion    string
	Signature    []byte
	Config       interface{}
}

type Plugin interface {
	Metadata() PluginMetadata
	Init() error
	PostLoad() error
	PreUnload() error
	Shutdown() error
	PreLoad(config []byte) error
	ManageConfig(config []byte) ([]byte, error)
	Execute(data interface{}) (interface{}, error)
}

type PluginStats struct {
	ExecutionCount     int64
	LastExecutionTime  time.Duration
	TotalExecutionTime time.Duration
}

var (
	pluginCache   sync.Map
	pluginCacheMu sync.Mutex
)

// LoadPlugin 优化:
// - 使用插件缓存提高加载效率
// - 使用双重检查锁定模式减少锁竞争
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
		return nil, errors.Wrapf(err, "打开插件失败: %s", path)
	}

	symPlugin, err := p.Lookup(PluginSymbol)
	if err != nil {
		return nil, errors.Wrapf(err, "查找插件符号失败: %s", path)
	}

	plugin, ok := symPlugin.(Plugin)
	if !ok {
		return nil, errors.Wrapf(ErrInvalidPluginInterface, "插件接口无效: %s", path)
	}

	pluginCache.Store(path, plugin)

	return plugin, nil
}

const PluginSymbol = "Plugin"
