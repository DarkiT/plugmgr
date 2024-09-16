package pluginmanager

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"path/filepath"
	"plugin"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkit/plugins/copier"
	"github.com/darkit/slog"
)

type Manager struct {
	mu            sync.RWMutex
	logger        *slog.Logger
	plugins       map[string]*lazyPlugin
	config        *Config
	dependencies  map[string][]string
	stats         map[string]*PluginStats
	eventBus      *EventBus
	sandbox       Sandbox
	publicKeyPath string
	pluginDir     string
}

type lazyPlugin struct {
	path   string
	loaded Plugin
}

func (lp *lazyPlugin) load() error {
	if lp.loaded == nil {
		p, err := plugin.Open(lp.path)
		if err != nil {
			return fmt.Errorf("打开插件失败: %w", err)
		}

		symPlugin, err := p.Lookup(PluginSymbol)
		if err != nil {
			return fmt.Errorf("查找插件符号失败: %w", err)
		}

		plugin, ok := symPlugin.(Plugin)
		if !ok {
			return fmt.Errorf("无效的插件接口")
		}

		lp.loaded = plugin
	}
	return nil
}

func NewManager(pluginDir, configPath string, publicKeyPath ...string) (*Manager, error) {
	config, err := LoadConfig(configPath, pluginDir)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	sandboxDir := filepath.Join(pluginDir, "sandbox")

	m := &Manager{
		plugins:      make(map[string]*lazyPlugin),
		config:       config,
		dependencies: make(map[string][]string),
		stats:        make(map[string]*PluginStats),
		eventBus:     NewEventBus(),
		sandbox:      NewLinuxSandbox(sandboxDir),
		logger:       slog.Default("plugins"),
		pluginDir:    pluginDir,
	}

	if len(publicKeyPath) > 0 {
		m.publicKeyPath = publicKeyPath[0]
	}

	// 如果配置为空，自动加载所有插件
	if len(m.config.Enabled) == 0 {
		if err := m.loadAllPlugins(); err != nil {
			return nil, fmt.Errorf("加载所有插件失败: %w", err)
		}
	}

	return m, nil
}

func (m *Manager) LoadPlugin(path string) error {
	return m.LoadPluginWithData(path)
}

func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return ErrPluginNotFound
	}

	if err := plugin.loaded.PreUnload(); err != nil {
		return fmt.Errorf("%s 的预卸载钩子失败: %w", name, err)
	}

	if err := plugin.loaded.Shutdown(); err != nil {
		return fmt.Errorf("%s 的关闭失败: %w", name, err)
	}

	delete(m.plugins, name)
	delete(m.dependencies, name)
	delete(m.stats, name)

	m.eventBus.Publish(PluginUnloadedEvent{PluginName: name})
	m.logger.Info("插件已卸载", slog.String("plugin", name))

	return nil
}

func (m *Manager) ExecutePlugin(name string, data any) (any, error) {
	return ExecutePluginGeneric[any, any](m, name, data)
}

func (m *Manager) HotReload(name string, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldPlugin, ok := m.plugins[name]
	if !ok {
		return ErrPluginNotFound
	}

	if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
		return fmt.Errorf("验证新插件签名失败: %w", err)
	}

	newLazyPlugin := &lazyPlugin{path: path}
	if err := newLazyPlugin.load(); err != nil {
		return fmt.Errorf("加载 %s 的新版本失败: %w", name, err)
	}

	newPlugin := newLazyPlugin.loaded

	metadata := newPlugin.Metadata()
	for dep, constraint := range metadata.Dependencies {
		if err := m.checkDependency(dep, constraint); err != nil {
			return fmt.Errorf("%s 新版本的依赖检查失败: %w", name, err)
		}
	}

	if err := newPlugin.Init(); err != nil {
		return fmt.Errorf("%s 新版本的初始化失败: %w", name, err)
	}

	if err := oldPlugin.loaded.PreUnload(); err != nil {
		m.logger.Warn("旧版本的预卸载钩子失败", slog.String("plugin", name), slog.Any("error", err))
	}
	if err := oldPlugin.loaded.Shutdown(); err != nil {
		m.logger.Warn("旧版本的关闭失败", slog.String("plugin", name), slog.Any("error", err))
	}

	m.plugins[name] = newLazyPlugin
	m.dependencies[name] = make([]string, 0, len(metadata.Dependencies))
	for dep := range metadata.Dependencies {
		m.dependencies[name] = append(m.dependencies[name], dep)
	}

	m.eventBus.Publish(PluginHotReloadedEvent{PluginName: name})
	m.logger.Info("插件热重载完成", slog.String("plugin", name))

	return nil
}

func (m *Manager) ManagePluginConfig(name string, config any) (any, error) {
	if config == nil {
		return ManagePluginConfigGeneric[any](m, name, nil)
	}
	return ManagePluginConfigGeneric(m, name, &config)
}

func (m *Manager) EnablePlugin(name string) error {
	if err := m.config.EnablePlugin(name); err != nil {
		return err
	}
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	return m.LoadPlugin(filepath.Join(m.pluginDir, name+".so"))
}

func (m *Manager) DisablePlugin(name string) error {
	if err := m.config.DisablePlugin(name); err != nil {
		return err
	}
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	return m.UnloadPlugin(name)
}

func (m *Manager) LoadEnabledPlugins(pluginDir string) error {
	enabled := m.config.EnabledPlugins()
	for _, name := range enabled {
		path := filepath.Join(pluginDir, name+".so")
		if err := m.LoadPlugin(path); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ListPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		plugins = append(plugins, name)
	}
	return plugins
}

func (m *Manager) GetPluginStats(name string) (*PluginStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats, ok := m.stats[name]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return stats, nil
}

func (m *Manager) SubscribeToEvent(eventName string, handler EventHandler) {
	m.eventBus.Subscribe(eventName, handler)
}

func (m *Manager) LoadPluginWithData(path string, data ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginName := filepath.Base(path)
	pluginName = strings.TrimSuffix(pluginName, ".so")

	if _, exists := m.plugins[pluginName]; exists {
		return fmt.Errorf("插件 %s 已加载", pluginName)
	}

	// 验证插件签名（如果启用）
	if m.publicKeyPath != "" {
		if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
			return fmt.Errorf("验证插件签名失败: %w", err)
		}
	}

	lazyPlug := &lazyPlugin{path: path}
	if err := lazyPlug.load(); err != nil {
		return fmt.Errorf("加载插件 %s 失败: %w", pluginName, err)
	}

	plugin := lazyPlug.loaded

	// 检查是否有保存的配置
	savedConfigBytes, hasSavedConfig := m.config.PluginConfigs[pluginName]

	var configToUse interface{}
	if hasSavedConfig {
		decoder := gob.NewDecoder(bytes.NewReader(savedConfigBytes))
		if err := decoder.Decode(&configToUse); err != nil {
			return fmt.Errorf("解码保存的插件配置失败: %w", err)
		}
	} else if len(data) > 0 {
		configToUse = data[0]
	}

	// 调用插件的 PreLoad 方法
	if err := plugin.PreLoad(configToUse); err != nil {
		return fmt.Errorf("%s 的预加载钩子失败: %w", pluginName, err)
	}

	// 调用插件的 Init 方法
	if err := plugin.Init(); err != nil {
		return fmt.Errorf("%s 的初始化失败: %w", pluginName, err)
	}

	// 调用插件的 PostLoad 方法
	if err := plugin.PostLoad(); err != nil {
		return fmt.Errorf("%s 的后加载钩子失败: %w", pluginName, err)
	}

	metadata := plugin.Metadata()

	// 更新配置
	if configToUse != nil {
		metadata.Config = configToUse
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		if err := encoder.Encode(configToUse); err != nil {
			return fmt.Errorf("序列化插件配置失败: %w", err)
		}
		m.config.PluginConfigs[pluginName] = buf.Bytes()
	}

	// 检查并记录插件依赖
	m.dependencies[pluginName] = make([]string, 0, len(metadata.Dependencies))
	for dep, constraint := range metadata.Dependencies {
		m.dependencies[pluginName] = append(m.dependencies[pluginName], dep)
		if err := m.checkDependency(dep, constraint); err != nil {
			delete(m.plugins, pluginName)
			delete(m.stats, pluginName)
			delete(m.dependencies, pluginName)
			return fmt.Errorf("%s 的依赖检查失败: %w", pluginName, err)
		}
	}

	// 将插件添加到已加载插件列表
	m.plugins[pluginName] = lazyPlug
	m.stats[pluginName] = &PluginStats{}

	// 保存更新后的配置
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	// 发布插件加载事件
	m.eventBus.Publish(PluginLoadedEvent{PluginName: pluginName})

	m.logger.Info("插件已加载",
		slog.String("plugin", pluginName),
		slog.String("version", metadata.Version),
		slog.Any("config", metadata.Config))

	return nil
}

func (m *Manager) GetPluginConfig(name string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, ErrPluginNotFound
	}

	metadata := plugin.loaded.Metadata()
	return metadata.Config, nil
}

func (m *Manager) checkDependency(depName, constraint string) error {
	depPlugin, exists := m.plugins[depName]
	if !exists {
		return fmt.Errorf("缺少依赖: %s", depName)
	}

	if err := depPlugin.load(); err != nil {
		return fmt.Errorf("加载依赖 %s 失败: %w", depName, err)
	}

	depVersion := depPlugin.loaded.Metadata().Version
	if !isVersionCompatible(depVersion, constraint) {
		return fmt.Errorf("依赖 %s 的版本不兼容: 需要 %s, 得到 %s", depName, constraint, depVersion)
	}

	return nil
}

func (m *Manager) getPluginConfigType(pluginName string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[pluginName]
	if !exists {
		return nil, ErrPluginNotFound
	}

	if err := plugin.load(); err != nil {
		return nil, fmt.Errorf("加载插件 %s 失败: %w", pluginName, err)
	}

	return plugin.loaded.ConfigType(), nil
}

func ManagePluginConfigGenericNew[T any](m *Manager, name string, config *T) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var zero T
	plugin, exists := m.plugins[name]
	if !exists {
		return zero, ErrPluginNotFound
	}

	if err := plugin.load(); err != nil {
		return zero, fmt.Errorf("加载插件 %s 失败: %w", name, err)
	}

	var updatedConfig interface{}
	var err error

	if config == nil {
		updatedConfig, err = plugin.loaded.ManageConfig(nil)
	} else {
		updatedConfig, err = plugin.loaded.ManageConfig(*config)
	}

	if err != nil {
		return zero, fmt.Errorf("更新插件 %s 的配置失败: %w", name, err)
	}

	// 尝试将 updatedConfig 转换为 T 类型
	typedConfig, ok := updatedConfig.(T)
	if !ok {
		return zero, fmt.Errorf("插件 %s 返回的配置类型不匹配", name)
	}

	// 如果 config 不为 nil，则复制配置
	if config != nil {
		if err = copier.Copy(config, &typedConfig); err != nil {
			return zero, fmt.Errorf("插件 %s 返回的配置类型不匹配: %w", name, err)
		}

		// 序列化配置
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		if err := encoder.Encode(*config); err != nil {
			return zero, fmt.Errorf("序列化插件配置失败: %w", err)
		}

		// 保存配置
		m.config.PluginConfigs[name] = buf.Bytes()

		if err := m.config.Save(); err != nil {
			return zero, fmt.Errorf("保存配置失败: %w", err)
		}

		m.logger.Info("插件配置已更新", slog.String("plugin", name))
	}

	return typedConfig, nil
}

func ManagePluginConfigGeneric[T any](m *Manager, name string, config *T) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var zero T
	plugin, exists := m.plugins[name]
	if !exists {
		return zero, ErrPluginNotFound
	}

	if err := plugin.load(); err != nil {
		return zero, fmt.Errorf("加载插件 %s 失败: %w", name, err)
	}

	var updatedConfig interface{}
	var err error

	if config == nil {
		// 如果config为nil,获取当前配置
		updatedConfig, err = plugin.loaded.ManageConfig(nil)
	} else {
		// 否则,更新配置
		updatedConfig, err = plugin.loaded.ManageConfig(*config)
	}

	if err != nil {
		return zero, fmt.Errorf("更新插件 %s 的配置失败: %w", name, err)
	}

	//copier.Copy(target, source)
	//target：复制到的目标对象。
	//source：复制自的源对象。
	//if err = copier.Copy(&updatedConfig, &config); err != nil {
	//	return zero, fmt.Errorf("插件 %s 返回的配置类型不匹配", name)
	//}
	// 尝试将updatedConfig转换为T类型
	typedConfig, ok := updatedConfig.(T)
	if !ok {
		return zero, fmt.Errorf("插件 %s 返回的配置类型不匹配", name)
	}

	// 如果提供了新的配置,则保存
	if config != nil {
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		if err := encoder.Encode(typedConfig); err != nil {
			return zero, fmt.Errorf("序列化插件配置失败: %w", err)
		}

		m.config.PluginConfigs[name] = buf.Bytes()

		if err := m.config.Save(); err != nil {
			return zero, fmt.Errorf("保存配置失败: %w", err)
		}

		m.logger.Info("插件配置已更新", slog.String("plugin", name))
	}

	return updatedConfig, nil
}

func ExecutePluginGeneric[T any, R any](m *Manager, name string, data T) (R, error) {
	m.mu.RLock()
	plugin, exists := m.plugins[name]
	stats := m.stats[name]
	m.mu.RUnlock()

	var zero R
	if !exists {
		return zero, ErrPluginNotFound
	}

	if err := m.sandbox.Enable(); err != nil {
		return zero, fmt.Errorf("为 %s 启用沙箱失败: %w", name, err)
	}
	defer m.sandbox.Disable()

	if err := plugin.load(); err != nil {
		return zero, fmt.Errorf("加载插件 %s 失败: %w", name, err)
	}

	start := time.Now()
	result, err := plugin.loaded.Execute(data)
	executionTime := time.Since(start)

	m.mu.Lock()
	stats.ExecutionCount++
	stats.LastExecutionTime = executionTime
	stats.TotalExecutionTime += executionTime
	m.mu.Unlock()

	if err != nil {
		return zero, fmt.Errorf("%s 的执行失败: %w", name, err)
	}

	m.logger.Info("插件已执行", slog.String("plugin", name), slog.Duration("duration", executionTime))

	// 尝试将结果转换为所需的类型
	typedResult, ok := result.(R)
	if !ok {
		return zero, fmt.Errorf("插件 %s 返回的结果类型不匹配", name)
	}

	return typedResult, nil
}

func isVersionCompatible(currentVersion, constraint string) bool {
	parts := strings.Split(constraint, " ")
	if len(parts) != 2 {
		return false
	}

	operator := parts[0]
	requiredVersion := parts[1]

	switch operator {
	case ">=":
		return compareVersions(currentVersion, requiredVersion) >= 0
	case ">":
		return compareVersions(currentVersion, requiredVersion) > 0
	case "<=":
		return compareVersions(currentVersion, requiredVersion) <= 0
	case "<":
		return compareVersions(currentVersion, requiredVersion) < 0
	case "==":
		return compareVersions(currentVersion, requiredVersion) == 0
	default:
		return false
	}
}

func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])

		if n1 < n2 {
			return -1
		} else if n1 > n2 {
			return 1
		}
	}

	if len(parts1) < len(parts2) {
		return -1
	} else if len(parts1) > len(parts2) {
		return 1
	}

	return 0
}
