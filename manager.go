package pluginmanager

import (
	"fmt"
	"path/filepath"
	"plugin"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/darkit/slog"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/sync/errgroup"
)

// Manager 结构体优化:
// - 使用 sync.Map 替代普通 map,提高并发安全性
type Manager struct {
	plugins       sync.Map
	config        *Config
	dependencies  sync.Map
	stats         sync.Map
	eventBus      *EventBus
	sandbox       Sandbox
	publicKeyPath string
	pluginDir     string
	logger        *slog.Logger

	versionManager *VersionManager
	pluginMarket   *PluginMarket
}

type lazyPlugin struct {
	path   string
	loaded Plugin
	mu     sync.Mutex
}

// load 方法优化:
// - 使用互斥锁确保并发安全
func (lp *lazyPlugin) load() error {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	if lp.loaded == nil {
		p, err := plugin.Open(lp.path)
		if err != nil {
			return errors.Wrapf(err, "打开插件失败: %s", lp.path)
		}

		symPlugin, err := p.Lookup(PluginSymbol)
		if err != nil {
			return errors.Wrapf(err, "查找插件符号失败: %s", lp.path)
		}

		plugin, ok := symPlugin.(Plugin)
		if !ok {
			return errors.Wrapf(ErrInvalidPluginInterface, "无效的插件接口: %s", lp.path)
		}

		lp.loaded = plugin
	}
	return nil
}

// NewManager 优化:
// - 使用 errors.Wrap 提供更详细的错误信息
func NewManager(pluginDir, configPath string, publicKeyPath ...string) (*Manager, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("插件系统暂不支持Windows环境下运行")
	}
	config, err := LoadConfig(configPath, pluginDir)
	if err != nil {
		return nil, errors.Wrap(err, "加载配置失败")
	}

	sandboxDir := filepath.Join(pluginDir, "sandbox")

	m := &Manager{
		config:         config,
		eventBus:       NewEventBus(),
		sandbox:        NewSandbox(sandboxDir),
		versionManager: NewVersionManager(),
		pluginMarket:   NewPluginMarket(),
		logger:         slog.Default("plugins"),
		pluginDir:      pluginDir,
	}

	if len(publicKeyPath) > 0 {
		m.publicKeyPath = publicKeyPath[0]
	}

	if len(m.config.Enabled) == 0 {
		if err := m.loadAllPlugins(); err != nil {
			return nil, errors.Wrap(err, "加载所有插件失败")
		}
	}

	return m, nil
}

// LoadPlugin 优化:
// - 使用 sync.Map 替代普通 map
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func (m *Manager) LoadPlugin(path string) error {
	pluginName := filepath.Base(path)
	pluginName = strings.TrimSuffix(pluginName, ".so")

	if _, loaded := m.plugins.LoadOrStore(pluginName, &lazyPlugin{path: path}); loaded {
		return errors.Errorf("插件 %s 已加载", pluginName)
	}

	if m.publicKeyPath != "" {
		if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
			return errors.Wrap(err, "验证插件签名失败")
		}
	}

	plugin, _ := m.plugins.Load(pluginName)
	lazyPlug := plugin.(*lazyPlugin)

	if err := lazyPlug.load(); err != nil {
		return errors.Wrapf(err, "加载插件 %s 失败", pluginName)
	}

	configToUse, err := m.loadPluginConfig(pluginName)
	if err != nil {
		return errors.Wrap(err, "加载插件配置失败")
	}

	if err := lazyPlug.loaded.PreLoad(configToUse); err != nil {
		return errors.Wrapf(err, "%s 的预加载钩子失败", pluginName)
	}

	if err := lazyPlug.loaded.Init(); err != nil {
		return errors.Wrapf(err, "%s 的初始化失败", pluginName)
	}

	if err := lazyPlug.loaded.PostLoad(); err != nil {
		return errors.Wrapf(err, "%s 的后加载钩子失败", pluginName)
	}

	metadata := lazyPlug.loaded.Metadata()

	if configToUse != nil {
		metadata.Config = configToUse
		m.config.PluginConfigs[pluginName] = configToUse
	}

	if err := m.checkDependencies(pluginName, metadata.Dependencies); err != nil {
		return errors.Wrap(err, "检查插件依赖失败")
	}

	m.stats.Store(pluginName, &PluginStats{})

	if err := m.config.Save(); err != nil {
		return errors.Wrap(err, "保存配置失败")
	}

	m.eventBus.Publish(PluginLoadedEvent{PluginName: pluginName})

	m.logger.Info("插件已加载", "plugin", pluginName, "version", metadata.Version)

	return nil
}

// UnloadPlugin 优化:
// - 使用 sync.Map 替代普通 map
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func (m *Manager) UnloadPlugin(name string) error {
	plugin, ok := m.plugins.Load(name)
	if !ok {
		return ErrPluginNotFound
	}

	lazyPlug := plugin.(*lazyPlugin)

	if err := lazyPlug.loaded.PreUnload(); err != nil {
		return errors.Wrapf(err, "%s 的预卸载钩子失败", name)
	}

	if err := lazyPlug.loaded.Shutdown(); err != nil {
		return errors.Wrapf(err, "%s 的关闭失败", name)
	}

	m.plugins.Delete(name)
	m.dependencies.Delete(name)
	m.stats.Delete(name)

	m.eventBus.Publish(PluginUnloadedEvent{PluginName: name})
	m.logger.Info("插件已卸载", "plugin", name)

	return nil
}

// ExecutePlugin 优化:
// - 使用类型断言来处理返回值
func (m *Manager) ExecutePlugin(name string, data interface{}) (interface{}, error) {
	result, err := ExecutePluginGeneric[interface{}, interface{}](m, name, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ExecutePluginString 执行返回字符串的插件
func (m *Manager) ExecutePluginString(name string, data interface{}) (string, error) {
	result, err := ExecutePluginGeneric[interface{}, string](m, name, data)
	if err != nil {
		return "", err
	}
	return result, nil
}

// ExecutePluginInt 执行返回整数的插件
func (m *Manager) ExecutePluginInt(name string, data interface{}) (int, error) {
	result, err := ExecutePluginGeneric[interface{}, int](m, name, data)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// ExecutePluginGeneric 优化:
// - 使用 sync.Map 替代普通 map
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func ExecutePluginGeneric[T any, R any](m *Manager, name string, data T) (R, error) {
	var zero R
	plugin, ok := m.plugins.Load(name)
	if !ok {
		return zero, ErrPluginNotFound
	}

	lazyPlug := plugin.(*lazyPlugin)

	if err := m.sandbox.Enable(); err != nil {
		return zero, errors.Wrapf(err, "为 %s 启用沙箱失败", name)
	}
	defer m.sandbox.Disable()

	if err := lazyPlug.load(); err != nil {
		return zero, errors.Wrapf(err, "加载插件 %s 失败", name)
	}

	start := time.Now()
	result, err := lazyPlug.loaded.Execute(data)
	executionTime := time.Since(start)

	m.updateStats(name, executionTime)

	if err != nil {
		return zero, errors.Wrapf(err, "%s 的执行失败", name)
	}

	m.logger.Info("插件已执行", "plugin", name, "duration", executionTime)

	typedResult, ok := result.(R)
	if !ok {
		return zero, errors.Errorf("插件 %s 返回的结果类型不匹配", name)
	}

	return typedResult, nil
}

// updateStats 优化:
// - 使用原子操作更新统计信息,提高并发安全性
func (m *Manager) updateStats(name string, executionTime time.Duration) {
	if stats, ok := m.stats.Load(name); ok {
		s := stats.(*PluginStats)
		atomic.AddInt64(&s.ExecutionCount, 1)
		atomic.StoreInt64((*int64)(&s.LastExecutionTime), int64(executionTime))
		atomic.AddInt64((*int64)(&s.TotalExecutionTime), int64(executionTime))
	}
}

// HotReload 优化:
// - 使用 sync.Map 替代普通 map
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func (m *Manager) HotReload(name string, path string) error {
	oldPlugin, ok := m.plugins.Load(name)
	if !ok {
		return ErrPluginNotFound
	}

	if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
		return errors.Wrap(err, "验证新插件签名失败")
	}

	newLazyPlugin := &lazyPlugin{path: path}
	if err := newLazyPlugin.load(); err != nil {
		return errors.Wrapf(err, "加载 %s 的新版本失败", name)
	}

	newPlugin := newLazyPlugin.loaded

	metadata := newPlugin.Metadata()
	if err := m.checkDependencies(name, metadata.Dependencies); err != nil {
		return errors.Wrapf(err, "插件 %s 关联依赖检查未通过", name)
	}

	if err := newPlugin.Init(); err != nil {
		return errors.Wrapf(err, "%s 新版本的初始化失败", name)
	}

	oldLazyPlugin := oldPlugin.(*lazyPlugin)
	if err := oldLazyPlugin.loaded.PreUnload(); err != nil {
		m.logger.Warn("旧版本的预卸载钩子失败", "plugin", name, "error", err)
	}
	if err := oldLazyPlugin.loaded.Shutdown(); err != nil {
		m.logger.Warn("旧版本的关闭失败", "plugin", name, "error", err)
	}

	m.plugins.Store(name, newLazyPlugin)
	m.dependencies.Store(name, metadata.Dependencies)

	m.eventBus.Publish(PluginHotReloadedEvent{PluginName: name})
	m.logger.Info("插件热重载完成", "plugin", name)

	return nil
}

// ManagePluginConfig 优化:
// - 使用 sync.Map 替代普通 map
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func (m *Manager) ManagePluginConfig(name string, config interface{}) ([]byte, error) {
	plugin, ok := m.plugins.Load(name)
	if !ok {
		return nil, ErrPluginNotFound
	}

	lazyPlug := plugin.(*lazyPlugin)

	if err := lazyPlug.load(); err != nil {
		return nil, errors.Wrapf(err, "加载插件 %s 失败", name)
	}

	serializer, err := Serializer(config)
	if err != nil {
		return nil, errors.Wrap(err, "序列化配置失败")
	}

	updatedConfig, err := lazyPlug.loaded.ManageConfig(serializer)
	if err != nil {
		return nil, errors.Wrapf(err, "更新插件 %s 的配置失败", name)
	}

	if config != nil {
		m.config.mu.Lock()
		m.config.PluginConfigs[name] = updatedConfig
		m.config.mu.Unlock()

		if err := m.config.Save(); err != nil {
			return nil, errors.Wrap(err, "保存配置失败")
		}
	}

	m.logger.Info("插件配置已更新", "plugin", name)
	return updatedConfig, nil
}

// EnablePlugin 优化:
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func (m *Manager) EnablePlugin(name string) error {
	if err := m.config.EnablePlugin(name); err != nil {
		return errors.Wrapf(err, "启用插件 %s 失败", name)
	}
	if err := m.config.Save(); err != nil {
		return errors.Wrap(err, "保存配置失败")
	}
	return m.LoadPlugin(filepath.Join(m.pluginDir, name+".so"))
}

// DisablePlugin 优化:
// - 优化错误处理,使用 errors.Wrap 提供更多上下文
func (m *Manager) DisablePlugin(name string) error {
	if err := m.config.DisablePlugin(name); err != nil {
		return errors.Wrapf(err, "禁用插件 %s 失败", name)
	}
	if err := m.config.Save(); err != nil {
		return errors.Wrap(err, "保存配置失败")
	}
	return m.UnloadPlugin(name)
}

// LoadEnabledPlugins 优化:
// - 使用 errgroup 并发加载插件
func (m *Manager) LoadEnabledPlugins(pluginDir string) error {
	enabled := m.config.EnabledPlugins()

	var eg errgroup.Group
	for _, name := range enabled {
		name := name // 创建局部变量避免闭包问题
		eg.Go(func() error {
			path := filepath.Join(pluginDir, name+".so")
			return m.LoadPlugin(path)
		})
	}

	return eg.Wait()
}

// ListPlugins 优化:
// - 使用 sync.Map 替代普通 map
func (m *Manager) ListPlugins() []string {
	var plugins []string
	m.plugins.Range(func(key, value interface{}) bool {
		plugins = append(plugins, key.(string))
		return true
	})
	return plugins
}

// GetPluginStats 优化:
// - 使用 sync.Map 替代普通 map
func (m *Manager) GetPluginStats(name string) (*PluginStats, error) {
	stats, ok := m.stats.Load(name)
	if !ok {
		return nil, ErrPluginNotFound
	}
	return stats.(*PluginStats), nil
}

func (m *Manager) SubscribeToEvent(eventName string, handler EventHandler) {
	m.eventBus.Subscribe(eventName, handler)
}

func (m *Manager) loadPluginConfig(pluginName string, data ...interface{}) ([]byte, error) {
	m.config.mu.RLock()
	savedConfigBytes, hasSavedConfig := m.config.PluginConfigs[pluginName]
	m.config.mu.RUnlock()

	if hasSavedConfig {
		return savedConfigBytes, nil
	} else if len(data) > 0 {
		return Serializer(data[0])
	}

	return nil, nil
}

func (m *Manager) LoadPluginWithData(path string, data ...interface{}) error {
	pluginName := filepath.Base(path)
	pluginName = strings.TrimSuffix(pluginName, ".so")

	if _, loaded := m.plugins.LoadOrStore(pluginName, &lazyPlugin{path: path}); loaded {
		return errors.Errorf("插件 %s 已加载", pluginName)
	}

	if m.publicKeyPath != "" {
		if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
			return errors.Wrap(err, "验证插件签名失败")
		}
	}

	plugin, _ := m.plugins.Load(pluginName)
	lazyPlug := plugin.(*lazyPlugin)

	if err := lazyPlug.load(); err != nil {
		return errors.Wrapf(err, "加载插件 %s 失败", pluginName)
	}

	configToUse, err := m.loadPluginConfig(pluginName, data...)
	if err != nil {
		return errors.Wrap(err, "加载插件配置失败")
	}

	if err := lazyPlug.loaded.PreLoad(configToUse); err != nil {
		return errors.Wrapf(err, "%s 的预加载钩子失败", pluginName)
	}

	if err := lazyPlug.loaded.Init(); err != nil {
		return errors.Wrapf(err, "%s 的初始化失败", pluginName)
	}

	if err := lazyPlug.loaded.PostLoad(); err != nil {
		return errors.Wrapf(err, "%s 的后加载钩子失败", pluginName)
	}

	metadata := lazyPlug.loaded.Metadata()

	if configToUse != nil {
		metadata.Config = configToUse
		m.config.mu.Lock()
		m.config.PluginConfigs[pluginName] = configToUse
		m.config.mu.Unlock()
	}

	if err := m.checkDependencies(pluginName, metadata.Dependencies); err != nil {
		return errors.Wrap(err, "检查插件依赖失败")
	}

	m.stats.Store(pluginName, &PluginStats{})

	if err := m.config.Save(); err != nil {
		return errors.Wrap(err, "保存配置失败")
	}

	m.eventBus.Publish(PluginLoadedEvent{PluginName: pluginName})

	m.logger.Info("插件已加载", "plugin", pluginName, "version", metadata.Version)

	return nil
}

func (m *Manager) GetPluginConfig(name string) (interface{}, error) {
	plugin, ok := m.plugins.Load(name)
	if !ok {
		return nil, ErrPluginNotFound
	}

	lazyPlug := plugin.(*lazyPlugin)
	if err := lazyPlug.load(); err != nil {
		return nil, errors.Wrapf(err, "加载插件 %s 失败", name)
	}

	metadata := lazyPlug.loaded.Metadata()
	return metadata.Config, nil
}

func (m *Manager) checkDependencies(pluginName string, dependencies map[string]string) error {
	visited := make(map[string]bool)
	var checkDep func(string, string, []string) error

	checkDep = func(depName, constraint string, depChain []string) error {
		if visited[depName] {
			cycle := append(depChain, depName)
			return errors.Errorf("检测到循环依赖: %s", strings.Join(cycle, " -> "))
		}
		visited[depName] = true

		depPlugin, ok := m.plugins.Load(depName)
		if !ok {
			return errors.Errorf("缺少依赖: %s", depName)
		}

		lazyPlug := depPlugin.(*lazyPlugin)
		if err := lazyPlug.load(); err != nil {
			return errors.Wrapf(err, "加载依赖 %s 失败", depName)
		}

		depMetadata := lazyPlug.loaded.Metadata()
		if !isVersionCompatible(depMetadata.Version, constraint) {
			return errors.Errorf("依赖 %s 的版本不兼容: 需要 %s, 得到 %s", depName, constraint, depMetadata.Version)
		}

		for subDepName, subConstraint := range depMetadata.Dependencies {
			if err := checkDep(subDepName, subConstraint, append(depChain, depName)); err != nil {
				return err
			}
		}

		return nil
	}

	for depName, constraint := range dependencies {
		if err := checkDep(depName, constraint, []string{pluginName}); err != nil {
			return err
		}
	}

	return nil
}

func Serializer(data interface{}) ([]byte, error) {
	return msgpack.Marshal(data)
}

func Deserializer(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
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

func (m *Manager) loadAllPlugins() error {
	files, err := filepath.Glob(filepath.Join(m.pluginDir, "*.so"))
	if err != nil {
		return errors.Wrap(err, "读取插件目录失败")
	}

	var eg errgroup.Group
	for _, file := range files {
		file := file // 创建局部变量避免闭包问题
		eg.Go(func() error {
			pluginName := strings.TrimSuffix(filepath.Base(file), ".so")
			m.config.mu.Lock()
			m.config.Enabled[pluginName] = true
			m.config.mu.Unlock()

			return m.LoadPluginWithData(file)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return m.config.Save()
}

func (m *Manager) PublishPlugin(info PluginInfo) error {
	m.pluginMarket.AddPlugin(info)
	return nil
}

func (m *Manager) DownloadAndInstallPlugin(name, version string) error {
	// 这里应该实现从远程服务器下载插件的逻辑
	// 为了演示，我们假设插件已经在本地
	pluginPath := filepath.Join(m.pluginDir, fmt.Sprintf("%s_v%s.so", name, version))

	if err := m.LoadPlugin(pluginPath); err != nil {
		return err
	}

	m.versionManager.AddVersion(name, version)
	m.versionManager.SetActiveVersion(name, version)

	return nil
}

func (m *Manager) ListAvailablePlugins() []PluginInfo {
	return m.pluginMarket.ListPlugins()
}

func (m *Manager) HotUpdatePlugin(name, newVersion string) error {
	_, exists := m.versionManager.GetActiveVersion(name)
	if !exists {
		return errors.New("插件未激活")
	}

	newPath := filepath.Join(m.pluginDir, fmt.Sprintf("%s_v%s.so", name, newVersion))
	if err := m.HotReload(name, newPath); err != nil {
		return err
	}

	m.versionManager.SetActiveVersion(name, newVersion)
	return nil
}

func (m *Manager) RollbackPlugin(name, targetVersion string) error {
	currentVersion, exists := m.versionManager.GetActiveVersion(name)
	if !exists {
		return errors.New("插件未激活")
	}

	if currentVersion == targetVersion {
		return nil
	}

	targetPath := filepath.Join(m.pluginDir, fmt.Sprintf("%s_v%s.so", name, targetVersion))
	if err := m.HotReload(name, targetPath); err != nil {
		return err
	}

	m.versionManager.SetActiveVersion(name, targetVersion)
	return nil
}
