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

// Config 配置结构
type Config struct {
	path          string                 `msgpack:"-"`       // 配置文件路径
	mu            sync.RWMutex           `msgpack:"-"`       // 读写锁
	enabled       map[string]bool        `msgpack:"enabled"` // 私有化：插件启用状态
	pluginConfigs map[string]*PluginData `msgpack:"configs"` // 私有化：插件配置数据
}

// NewConfig 创建新的配置实例
func NewConfig(filename string) *Config {
	return &Config{
		path:          filename,
		enabled:       make(map[string]bool),
		pluginConfigs: make(map[string]*PluginData),
	}
}

// LoadConfig 加载配置文件
func LoadConfig(filename string, pluginDir ...string) (*Config, error) {
	if len(pluginDir) > 0 {
		filename = filepath.Join(pluginDir[0], filename)
	}

	config := NewConfig(filename)

	file, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, wrap(err, "读取配置文件失败")
	}

	if err := msgpack.Unmarshal(file, config); err != nil {
		return nil, wrap(err, "解析配置文件失败")
	}

	return config, nil
}

// Save 保存配置到文件
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := msgpack.Marshal(c)
	if err != nil {
		return wrap(err, "序列化配置失败")
	}

	return wrap(os.WriteFile(c.path, data, 0o644), "写入配置文件失败")
}

// GetPluginConfig 获取插件配置
func (c *Config) GetPluginConfig(name string) (*PluginData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, exists := c.pluginConfigs[name]
	return data, exists
}

// SetPluginConfig 设置插件配置
func (c *Config) SetPluginConfig(name string, config []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pluginConfigs[name] = &PluginData{
		Config:    config,
		UpdatedAt: time.Now(),
	}

	return c.Save()
}

// IsEnabled 检查插件是否启用
func (c *Config) IsEnabled(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled[name]
}

// SetEnabled 设置插件启用状态
func (c *Config) SetEnabled(name string, status bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enabled[name] = status
	return c.Save()
}

// GetEnabledPlugins 获取所有启用的插件
func (c *Config) GetEnabledPlugins() []string {
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
func (c *Config) GetPluginLastUpdated(name string) (time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, exists := c.pluginConfigs[name]; exists {
		return data.UpdatedAt, true
	}
	return time.Time{}, false
}

// GetPluginPermissions 获取插件权限
func (c *Config) GetPluginPermissions(name string) (*PluginPermission, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, exists := c.pluginConfigs[name]; exists && data.Permissions != nil {
		return data.Permissions, true
	}
	return nil, false
}

// SetPluginPermissions 设置插件权限
func (c *Config) SetPluginPermissions(name string, permissions *PluginPermission) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if data, exists := c.pluginConfigs[name]; exists {
		data.Permissions = permissions
		return c.Save()
	}
	return newError("plugin not found")
}
