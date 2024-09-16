// 版权所有 (C) 2024 Matt Dunleavy。保留所有权利。
// 本源代码的使用受 LICENSE 文件中的 MIT 许可证约束。

package pluginmanager

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/darkit/slog"
)

type Config struct {
	Enabled       map[string]bool
	PluginConfigs map[string][]byte // 使用 []byte 来存储序列化后的配置
	path          string
	mu            sync.RWMutex
}

func LoadConfig(filename string, pluginDir ...string) (config *Config, err error) {
	if len(pluginDir) > 0 {
		filename = filepath.Join(pluginDir[0], filename)
	}

	config = &Config{
		Enabled:       make(map[string]bool),
		PluginConfigs: make(map[string][]byte),
		path:          filename,
	}

	file, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	decoder := gob.NewDecoder(bytes.NewReader(file))
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (m *Manager) loadAllPlugins() error {
	files, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return fmt.Errorf("读取插件目录失败: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".so" {
			pluginName := strings.TrimSuffix(file.Name(), ".so")
			m.config.Enabled[pluginName] = true

			// 获取保存的配置（如果有）
			savedConfigBytes := m.config.PluginConfigs[pluginName]
			var savedConfig interface{}
			if len(savedConfigBytes) > 0 {
				decoder := gob.NewDecoder(bytes.NewReader(savedConfigBytes))
				if err := decoder.Decode(&savedConfig); err != nil {
					m.logger.Warn("解码保存的插件配置失败", slog.String("plugin", pluginName), slog.Any("error", err))
				}
			}

			if err := m.LoadPluginWithData(filepath.Join(m.pluginDir, file.Name()), savedConfig); err != nil {
				m.logger.Warn("加载插件失败", slog.String("plugin", pluginName), slog.Any("error", err))
			}
		}
	}

	// 保存新的配置
	return m.config.Save()
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(c); err != nil {
		return err
	}

	return os.WriteFile(c.path, buf.Bytes(), 0o644)
}

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
