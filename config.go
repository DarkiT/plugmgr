package plugmgr

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	msgpack "github.com/vmihailenco/msgpack/v5"
)

// PluginData 存储插件配置的详细信息
type PluginData struct {
	Config      []byte            `msgpack:"config"`     // 插件的配置数据
	UpdatedAt   time.Time         `msgpack:"updated_at"` // 最后更新时间
	Permissions *PluginPermission `msgpack:"permissions,omitempty"`
}

// config 配置结构
type config struct {
	path          string                 `msgpack:"-"`       // 配置文件路径
	mu            sync.RWMutex           `msgpack:"-"`       // 读写锁
	enabled       map[string]bool        `msgpack:"enabled"` // 私有化：插件启用状态
	pluginConfigs map[string]*PluginData `msgpack:"configs"` // 私有化：插件配置数据
}

// NewConfig 创建新的配置实例
func NewConfig(filename string) *config {
	return &config{
		path:          filename,
		enabled:       make(map[string]bool),
		pluginConfigs: make(map[string]*PluginData),
	}
}

// LoadConfig 加载配置文件
func LoadConfig(filename string, pluginDir ...string) (*config, error) {
	if len(pluginDir) > 0 {
		filename = filepath.Join(pluginDir[0], filename)
	}

	c := NewConfig(filename)

	file, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, wrap(err, "读取配置文件失败")
	}

	if err := msgpack.Unmarshal(file, c); err != nil {
		return nil, wrap(err, "解析配置文件失败")
	}

	return c, nil
}

// Save 保存配置到文件
func (c *config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := msgpack.Marshal(c)
	if err != nil {
		return wrap(err, "序列化配置失败")
	}

	return wrap(os.WriteFile(c.path, data, 0o644), "写入配置文件失败")
}

// GetPluginConfig 获取插件配置
func (c *config) GetPluginConfig(name string) (*PluginData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, exists := c.pluginConfigs[name]
	return data, exists
}

// SetPluginConfig 设置插件配置
func (c *config) SetPluginConfig(name string, config []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pluginConfigs[name] = &PluginData{
		Config:    config,
		UpdatedAt: time.Now(),
	}

	return c.Save()
}

// IsEnabled 检查插件是否启用
func (c *config) IsEnabled(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled[name]
}

// SetEnabled 设置插件启用状态
func (c *config) SetEnabled(name string, status bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enabled[name] = status
	return c.Save()
}

// GetEnabledPlugins 获取所有启用的插件
func (c *config) GetEnabledPlugins() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var enabled []string
	for name, isEnabled := range c.enabled {
		if isEnabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// GetPluginLastUpdated 获取插件配置最后更新时间
func (c *config) GetPluginLastUpdated(name string) (time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, exists := c.pluginConfigs[name]; exists {
		return data.UpdatedAt, true
	}
	return time.Time{}, false
}

// GetPluginPermissions 获取插件权限
func (c *config) GetPluginPermissions(name string) (*PluginPermission, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, exists := c.pluginConfigs[name]; exists && data.Permissions != nil {
		return data.Permissions, true
	}
	return nil, false
}

// SetPluginPermissions 设置插件权限
func (c *config) SetPluginPermissions(name string, permissions *PluginPermission) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if data, exists := c.pluginConfigs[name]; exists {
		data.Permissions = permissions
		return c.Save()
	}
	return newError("plugin not found")
}
