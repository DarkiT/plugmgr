package pluginmanager

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	msgpack "github.com/vmihailenco/msgpack/v5"
)

// Config 结构体优化:
// - 使用 sync.RWMutex 来保证并发安全
// - 使用 msgpack 替代 gob 进行序列化,提高效率
type Config struct {
	Enabled       map[string]bool
	PluginConfigs map[string][]byte
	path          string
	mu            sync.RWMutex
}

// LoadConfig 优化:
// - 使用 msgpack 进行反序列化
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func LoadConfig(filename string, pluginDir ...string) (*Config, error) {
	if len(pluginDir) > 0 {
		filename = filepath.Join(pluginDir[0], filename)
	}

	config := &Config{
		Enabled:       make(map[string]bool),
		PluginConfigs: make(map[string][]byte),
		path:          filename,
	}

	file, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, errors.Wrap(err, "读取配置文件失败")
	}

	if err := msgpack.Unmarshal(file, config); err != nil {
		return nil, errors.Wrap(err, "解析配置文件失败")
	}

	return config, nil
}

// Save 方法优化:
// - 使用 msgpack 进行序列化
// - 使用读写锁确保并发安全
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := msgpack.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "序列化配置失败")
	}

	return errors.Wrap(os.WriteFile(c.path, data, 0o644), "写入配置文件失败")
}

// EnablePlugin 和 DisablePlugin 方法优化:
// - 使用写锁确保并发安全
func (c *Config) EnablePlugin(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Enabled[name] = true
	return nil
}

func (c *Config) DisablePlugin(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Enabled[name] = false
	return nil
}

// EnabledPlugins 方法优化:
// - 使用读锁提高并发性能
func (c *Config) EnabledPlugins() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var enabled []string
	for name, isEnabled := range c.Enabled {
		if isEnabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}
