package plugmgr

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"plugin"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/sync/errgroup"
)

// PluginPermission 插件权限定义
type PluginPermission struct {
	AllowedActions map[string]bool // 允许的操作列表
	Roles          []string        // 角色列表
}

// Manager 插件管理器
//
//	功能:
//	- 管理插件的生命周期（加载、执行、卸载）
//	- 处理插件配置和依赖关系
//	- 提供插件权限控制
//	字段说明:
//	- plugins: 已加载的插件映射
//	- config: 插件配置管理器
//	- dependencies: 插件依赖关系映射
//	- stats: 插件执行统计信息
//	- eventBus: 事件总线，用于插件事件通知
//	- sandbox: 插件沙箱环境
//	- publicKeyPath: 插件签名验证公钥路径
//	- pluginDir: 插件目录路径
//	- logger: 日志记录器
//	- versionManager: 插件版本管理器
//	- pluginMarket: 插件市场接口
//	- permissions: 插件权限配置映射
type Manager struct {
	plugins       sync.Map // map[string]*lazyPlugin
	config        *config
	dependencies  sync.Map // map[string]map[string]string
	stats         sync.Map // map[string]*PluginStats
	eventBus      *eventBus
	sandbox       Sandbox
	publicKeyPath string
	pluginDir     string
	logger        Logger

	versionManager   *VersionManager
	pluginMarket     *PluginMarket
	permissions      sync.Map // map[string]*PluginPermission
	preloadedPlugins sync.Map
}

type lazyPlugin struct {
	path   string
	loaded Plugin
	mu     sync.Mutex
}

// load 加载插件实例
func (lp *lazyPlugin) load() error {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	if lp.loaded == nil {
		p, err := plugin.Open(lp.path)
		if err != nil {
			return wrapf(err, "打开插件失败: %s", lp.path)
		}

		symPlugin, err := p.Lookup(PluginSymbol)
		if err != nil {
			return wrapf(err, "查找插件符号失败: %s", lp.path)
		}

		pluginInfo, ok := symPlugin.(Plugin)
		if !ok {
			return wrapf(ErrInvalidPluginInterface, "无效的插件接口: %s", lp.path)
		}

		lp.loaded = pluginInfo
	}
	return nil
}

// NewManager 创建新的插件管理器实例
//
//	参数:
//	- pluginDir: 插件目录路径
//	- configPath: 配置文件路径
//	- publicKeyPath: 可选的公钥路径，用于验证插件签名
//	功能:
//	- 初始化插件管理器及其依赖组件
//	- 加载配置文件
//	- 设置沙箱环境
//	- 初始化事件总线、版本管理器和插件市场
//	- 如果未指定启用的插件，则加载所有插件
//	返回:
//	- *Manager: 插件管理器实例
//	- error: 初始化过程中的错误信息
func NewManager(pluginDir, configPath string, publicKeyPath ...string) (*Manager, error) {
	if runtime.GOOS == "windows" {
		return nil, newError("插件系统暂不支持Windows环境下运行")
	}
	config, err := LoadConfig(configPath, pluginDir)
	if err != nil {
		return nil, wrap(err, "加载配置失败")
	}

	sandboxDir := filepath.Join(pluginDir, "sandbox")

	m := &Manager{
		config:         config,
		eventBus:       newEventBus(),
		sandbox:        newSandbox(sandboxDir),
		versionManager: newVersionManager(),
		pluginMarket:   newPluginMarket(),
		logger:         &logger{logger: slog.Default()},
		pluginDir:      pluginDir,
	}

	if len(publicKeyPath) > 0 {
		m.publicKeyPath = publicKeyPath[0]
	}

	if len(m.config.enabled) == 0 {
		if err := m.loadAllPlugins(); err != nil {
			return nil, wrap(err, "加载所有插件失败")
		}
	}

	return m, nil
}

// SetLogger 设置日志记录器
//
//	logger: 日志记录器实例
func (m *Manager) SetLogger(logger Logger) {
	m.logger = logger
}

// PreloadPlugins 添加预加载方法
//
//	参数:
//	- names: 插件名称列表
//	返回:
//	- error: 预加载过程中的错误
//	功能:
//	- 预加载指定插件
//	- 不立即初始化，仅加载插件符号
func (m *Manager) PreloadPlugins(names []string) error {
	for _, name := range names {
		if err := m.preloadPlugin(name); err != nil {
			return err
		}
	}
	return nil
}

// LoadPlugin 加载一个插件
//
//	path: 插件文件的完整路径
//	功能:
//	- 验证插件签名(如果启用)
//	- 加载插件并初始化
//	- 设置默认权限
//	- 触发加载事件
func (m *Manager) LoadPlugin(path string) error {
	if m.publicKeyPath != "" {
		if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
			return wrap(err, "验证插件签名失败")
		}
	}

	pluginName := filepath.Base(path)
	pluginName = strings.TrimSuffix(pluginName, ".so")

	if _, loaded := m.plugins.LoadOrStore(pluginName, &lazyPlugin{path: path}); loaded {
		return newErrorf("插件 %s 已加载", pluginName)
	}

	pluginInfo, _ := m.plugins.Load(pluginName)
	lazyPlug := pluginInfo.(*lazyPlugin)

	if err := lazyPlug.load(); err != nil {
		return wrapf(err, "加载插件 %s 失败", pluginName)
	}

	configToUse, err := m.loadPluginConfig(pluginName)
	if err != nil {
		return wrap(err, "加载插件配置失败")
	}

	if err := lazyPlug.loaded.PreLoad(configToUse); err != nil {
		return wrapf(err, "%s 的预加载钩子失败", pluginName)
	}

	if err := lazyPlug.loaded.Init(); err != nil {
		return wrapf(err, "%s 的初始化失败", pluginName)
	}

	// 在插件初始化后触发事件
	m.eventBus.PublishAsync(Event{
		EventName: PluginInitialized,
		Data: EventData{
			Name: pluginName,
		},
	})

	if err := lazyPlug.loaded.PostLoad(); err != nil {
		return wrapf(err, "%s 的后加载钩子失败", pluginName)
	}

	metadata := lazyPlug.loaded.Metadata()

	if configToUse != nil {
		metadata.Config = configToUse
		err = m.config.SetPluginConfig(pluginName, configToUse)
		if err != nil {
			return wrap(err, "保存配置失败")
		}
	}

	if err := m.checkDependencies(pluginName, metadata.Dependencies); err != nil {
		return wrap(err, "检查插件依赖失败")
	}

	m.stats.Store(pluginName, &PluginStats{})

	// 使用 Notify 方法发布插件加载事件
	m.eventBus.PublishAsync(Event{
		EventName: PluginLoaded,
		Data: EventData{
			Name: pluginName,
		},
	})

	m.logger.Info("插件已加载", "plugin", pluginName, "version", metadata.Version)

	// 初始化默认权限
	m.SetPluginPermission(pluginName, &PluginPermission{
		AllowedActions: map[string]bool{
			"execute": true,  // 默认允许执行
			"read":    true,  // 默认允许读取
			"write":   false, // 默认禁止写入
			"admin":   false, // 默认禁止管理操作
		},
		Roles: []string{"user"}, // 默认用户角色
	})

	return nil
}

// UnloadPlugin 卸载指定的插件
//
//	name: 插件名称
//	功能:
//	- 执行插件的预卸载和关闭钩子
//	- 清理插件资源和权限
//	- 触发卸载事件
func (m *Manager) UnloadPlugin(name string) error {
	pluginInfo, ok := m.plugins.Load(name)
	if !ok {
		return ErrPluginNotFound
	}

	lazyPlug := pluginInfo.(*lazyPlugin)

	// 在插件卸载前触发事件
	m.eventBus.PublishAsync(Event{
		EventName: PluginPreUnload,
		Data: EventData{
			Name: name,
		},
	})

	if err := lazyPlug.loaded.PreUnload(); err != nil {
		return wrapf(err, "%s 的预卸载钩子失败", name)
	}

	if err := lazyPlug.loaded.Shutdown(); err != nil {
		return wrapf(err, "%s 的关闭失败", name)
	}

	m.plugins.Delete(name)
	m.dependencies.Delete(name)
	m.stats.Delete(name)

	m.eventBus.PublishAsync(Event{
		EventName: PluginUnloaded,
		Data: EventData{
			Name: name,
		},
	})
	m.logger.Info("插件已卸载", "plugin", name)

	// 清理插件权限
	m.RemovePluginPermission(name)

	return nil
}

// ExecutePlugin 执行插件并返回通用类型结果
//
//	name: 插件名称
//	data: 传递给插件的数据
//	功能:
//	- 在沙箱环境中执行插件
//	- 更新执行统计信息
//	- 返回任意类型的结果
func (m *Manager) ExecutePlugin(name string, data any) (any, error) {
	result, err := ExecutePluginGeneric[any, any](m, name, data)
	if err != nil {
		// 在插件执行错误时触发事件
		m.eventBus.PublishAsync(Event{
			EventName: PluginExecutionError,
			Data: EventData{
				Name:  name,
				Error: err,
			},
		})
		return nil, wrap(err, "执行插件失败")
	}

	// 在插件执行后触发事件
	m.eventBus.PublishAsync(Event{
		EventName: PluginExecuted,
		Data: EventData{
			Name: name,
			Data: result,
		},
	})
	return result, nil
}

// ExecutePluginString 执行插件并返回字符串结果
//
//	name: 插件名称
//	data: 传递给插件的数据
//	功能:
//	- 执行插件并确保返回字符串类型
func (m *Manager) ExecutePluginString(name string, data any) (string, error) {
	result, err := ExecutePluginGeneric[any, string](m, name, data)
	if err != nil {
		return "", err
	}
	return result, nil
}

// ExecutePluginInt 执行插件并返回整数结果
//
//	name: 插件名称
//	data: 传递给插件的数据
//	功能:
//	- 执行插件并确保返回整数类型
func (m *Manager) ExecutePluginInt(name string, data any) (int, error) {
	result, err := ExecutePluginGeneric[any, int](m, name, data)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// ExecutePluginGeneric 通用插件执行函数
//
//	类型参数:
//	- T: 输入数据类型
//	- R: 返回结果类型
//	参数:
//	- m: 插件管理器实例
//	- name: 插件名称
//	- data: 传递给插件的数据
//	功能:
//	- 在沙箱环境中安全执行插件
//	- 记录执行时间和统计信息
//	- 确保类型安全的结果转换
//	返回:
//	- R: 类型安全的执行结果
//	- error: 执行过程中的错误信息
func ExecutePluginGeneric[T any, R any](m *Manager, name string, data T) (R, error) {
	var zero R
	if !m.HasPermission(name, "execute") {
		return zero, newErrorf("插件 %s 没有执行权限", name)
	}

	pluginInfo, ok := m.plugins.Load(name)
	if !ok {
		return zero, wrapf(ErrPluginNotFound, "插件 %s 未找到", name)
	}

	lazyPlug := pluginInfo.(*lazyPlugin)

	m.logger.Info("开始执行插件",
		"plugin", name,
		"dataType", fmt.Sprintf("%T", data))

	if err := m.sandbox.Enable(); err != nil {
		m.logger.Error("启用沙箱失败",
			"plugin", name,
			"error", err)
		return zero, wrapf(err, "为 %s 启用沙箱失败", name)
	}
	defer func() {
		if err := m.sandbox.Disable(); err != nil {
			m.logger.Error("禁用沙箱失败",
				"plugin", name,
				"error", err)
		}
	}()

	if err := lazyPlug.load(); err != nil {
		return zero, wrapf(err, "加载插件 %s 失败", name)
	}

	start := time.Now()
	result, err := lazyPlug.loaded.Execute(data)
	executionTime := time.Since(start)

	m.updateStats(name, executionTime)

	if err != nil {
		m.logger.Error("插件执行失败",
			"plugin", name,
			"error", err,
			"duration", executionTime)
		return zero, wrapf(err, "%s 的执行失败", name)
	}

	m.logger.Info("插件执行完成",
		"plugin", name,
		"duration", executionTime,
		"resultType", fmt.Sprintf("%T", result))

	typedResult, ok := result.(R)
	if !ok {
		return zero, newErrorf("插件 %s 返回的结果类型不匹配: 期望 %T, 得到 %T", name, zero, result)
	}

	return typedResult, nil
}

func (m *Manager) updateStats(name string, executionTime time.Duration) {
	if stats, ok := m.stats.Load(name); ok {
		s := stats.(*PluginStats)
		atomic.AddInt64(&s.ExecutionCount, 1)
		atomic.StoreInt64((*int64)(&s.LastExecutionTime), int64(executionTime))
		atomic.AddInt64((*int64)(&s.TotalExecutionTime), int64(executionTime))
	}
}

// HotReload 热重载插件
//
//	name: 插件名称
//	path: 新插件文件的路径
//	功能:
//	- 验证新插件签名
//	- 保持原有配置的情况下更新插件
//	- 触发热重载事件
func (m *Manager) HotReload(name string, path string) error {
	if m.publicKeyPath != "" {
		if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
			return wrap(err, "验证新插件签名失败")
		}
	}

	oldPlugin, ok := m.plugins.Load(name)
	if !ok {
		return ErrPluginNotFound
	}

	newLazyPlugin := &lazyPlugin{path: path}
	if err := newLazyPlugin.load(); err != nil {
		return wrapf(err, "加载 %s 的新版本失败", name)
	}

	newPlugin := newLazyPlugin.loaded

	metadata := newPlugin.Metadata()
	if err := m.checkDependencies(name, metadata.Dependencies); err != nil {
		return wrapf(err, "插件 %s 关联依赖检查未通过", name)
	}

	if err := newPlugin.Init(); err != nil {
		return wrapf(err, "%s 新版本的初始化失败", name)
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

	m.eventBus.PublishAsync(Event{
		EventName: PluginHotReloaded,
		Data: EventData{
			Name: name,
		},
	})
	m.logger.Info("插件热重载完成", "plugin", name)

	return nil
}

// ConfigUpdated 管理插件配置
//
//	name: 插件名称
//	config: 新的配置数据
//	功能:
//	- 序列化配置数据
//	- 更新插件配置
//	- 保存配置到持久化存储
func (m *Manager) ConfigUpdated(name string, config any) ([]byte, error) {
	pluginInfo, ok := m.plugins.Load(name)
	if !ok {
		return nil, ErrPluginNotFound
	}

	lazyPlug := pluginInfo.(*lazyPlugin)

	if err := lazyPlug.load(); err != nil {
		return nil, wrapf(err, "加载插件 %s 失败", name)
	}

	serializer, err := Serializer(config)
	if err != nil {
		return nil, wrap(err, "序列化配置失败")
	}

	updatedConfig, err := lazyPlug.loaded.ConfigUpdated(serializer)
	if err != nil {
		return nil, wrapf(err, "更新插件 %s 的配置失败", name)
	}

	if config != nil {
		if err = m.config.SetPluginConfig(name, updatedConfig); err != nil {
			return nil, wrap(err, "保存配置失败")
		}
		// 在插件配置更新后触发事件
		m.eventBus.PublishAsync(Event{
			EventName: PluginConfigUpdated,
			Data: EventData{
				Name: name,
				Data: updatedConfig,
			},
		})
	}

	m.logger.Info("插件配置已更新", "plugin", name)
	return updatedConfig, nil
}

// EnablePlugin 启用插件
//
//	name: 插件名称
//	功能:
//	- 更新插件启用状态
//	- 加载插件
func (m *Manager) EnablePlugin(name string) error {
	if err := m.config.SetEnabled(name, true); err != nil {
		return wrapf(err, "启用插件 %s 失败", name)
	}
	return m.LoadPlugin(filepath.Join(m.pluginDir, name+".so"))
}

// DisablePlugin 禁用插件
//
//	name: 插件名称
//	功能:
//	- 更新插件禁用状态
//	- 卸载插件
func (m *Manager) DisablePlugin(name string) error {
	if err := m.config.SetEnabled(name, false); err != nil {
		return wrapf(err, "禁用插件 %s 失败", name)
	}
	return m.UnloadPlugin(name)
}

// LoadEnabledPlugins 加载所有启用的插件
//
//	pluginDir: 插件目录路径
//	功能:
//	- 并发加载所有启用的插件
func (m *Manager) LoadEnabledPlugins(pluginDir string) error {
	enabled := m.config.GetEnabledPlugins()

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

// ListPlugins 列出所有已加载的插件
//
//	功能:
//	- 返回当前已加载的插件名称列表
func (m *Manager) ListPlugins() []string {
	var plugins []string
	m.plugins.Range(func(key, value any) bool {
		plugins = append(plugins, key.(string))
		return true
	})
	return plugins
}

// GetPluginStats 获取插件统计信息
//
//	name: 插件名称
//	功能:
//	- 返回插件的执行统计信息
func (m *Manager) GetPluginStats(name string) (*PluginStats, error) {
	stats, ok := m.stats.Load(name)
	if !ok {
		return nil, ErrPluginNotFound
	}
	return stats.(*PluginStats), nil
}

// SubscribeToEvent 订阅插件事件
// 参数:
// - eventName: 事件名称
// - handler: 事件处理函数
// 功能:
// - 注册事件处理器
// - 当指定事件发生时触发处理函数
func (m *Manager) SubscribeToEvent(eventName string, handler EventHandler) {
	m.eventBus.Subscribe(eventName, handler)
}

func (m *Manager) loadPluginConfig(pluginName string, data ...any) ([]byte, error) {
	if config, exists := m.config.GetPluginConfig(pluginName); exists {
		return config.Config, nil
	} else if len(data) > 0 {
		return Serializer(data[0])
	}
	return nil, nil
}

// LoadPluginWithData 加载插件并设置初始数据
//
//	path: 插件文件路径
//	data: 可选的初始配置数据
//	功能:
//	- 加载插件
//	- 设置初始配置
//	- 执行完整的插件初始化流程
func (m *Manager) LoadPluginWithData(path string, data ...any) error {
	if m.publicKeyPath != "" {
		if err := m.VerifyPluginSignature(path, m.publicKeyPath); err != nil {
			return wrap(err, "验证插件签名失败")
		}
	}

	pluginName := filepath.Base(path)
	pluginName = strings.TrimSuffix(pluginName, ".so")

	if _, loaded := m.plugins.LoadOrStore(pluginName, &lazyPlugin{path: path}); loaded {
		return newErrorf("插件 %s 已加载", pluginName)
	}

	pluginInfo, _ := m.plugins.Load(pluginName)
	lazyPlug := pluginInfo.(*lazyPlugin)

	if err := lazyPlug.load(); err != nil {
		return wrapf(err, "加载插件 %s 失败", pluginName)
	}

	configToUse, err := m.loadPluginConfig(pluginName, data...)
	if err != nil {
		return wrap(err, "加载插件配置失败")
	}

	if err = lazyPlug.loaded.PreLoad(configToUse); err != nil {
		return wrapf(err, "%s 的预加载钩子失败", pluginName)
	}

	if err = lazyPlug.loaded.Init(); err != nil {
		return wrapf(err, "%s 的初始化失败", pluginName)
	}

	if err = lazyPlug.loaded.PostLoad(); err != nil {
		return wrapf(err, "%s 的后加载钩子失败", pluginName)
	}

	metadata := lazyPlug.loaded.Metadata()

	if configToUse != nil {
		metadata.Config = configToUse
		if err = m.config.SetPluginConfig(pluginName, configToUse); err != nil {
			return wrap(err, "保存插件配置失败")
		}
	}

	if err = m.checkDependencies(pluginName, metadata.Dependencies); err != nil {
		return wrap(err, "检查插件依赖失败")
	}

	m.stats.Store(pluginName, &PluginStats{})

	m.eventBus.PublishAsync(Event{
		EventName: PluginLoaded,
		Data: EventData{
			Name: pluginName,
		},
	})

	m.logger.Info("插件已加载", "plugin", pluginName, "version", metadata.Version)

	return nil
}

// GetPluginConfig 获取插件配置
//
//	name: 插件名称
//	功能:
//	- 返回插件的当前配置
func (m *Manager) GetPluginConfig(name string) (*PluginData, error) {
	pluginInfo, ok := m.plugins.Load(name)
	if !ok {
		return nil, ErrPluginNotFound
	}

	lazyPlug := pluginInfo.(*lazyPlugin)
	if err := lazyPlug.load(); err != nil {
		return nil, wrapf(err, "加载插件 %s 失败", name)
	}

	if config, exists := m.config.GetPluginConfig(name); exists {
		return config, nil
	}

	return nil, nil
}

func (m *Manager) checkDependencies(pluginName string, dependencies map[string]string) error {
	visited := make(map[string]bool)
	var checkDep func(string, string, []string) error

	checkDep = func(depName, constraint string, depChain []string) error {
		if visited[depName] {
			cycle := append(depChain, depName)
			return newErrorf("检测到循环依赖: %s", strings.Join(cycle, " -> "))
		}
		visited[depName] = true

		depPlugin, ok := m.plugins.Load(depName)
		if !ok {
			return newErrorf("缺少依赖: %s", depName)
		}

		lazyPlug := depPlugin.(*lazyPlugin)
		if err := lazyPlug.load(); err != nil {
			return wrapf(err, "加载依赖 %s 失败", depName)
		}

		depMetadata := lazyPlug.loaded.Metadata()
		if !isVersionCompatible(depMetadata.Version, constraint) {
			return newErrorf("依赖 %s 的版本不兼容: 需要 %s, 得到 %s", depName, constraint, depMetadata.Version)
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

func (m *Manager) loadAllPlugins() error {
	files, err := filepath.Glob(filepath.Join(m.pluginDir, "*.so"))
	if err != nil {
		return wrap(err, "读取插件目录失败")
	}

	var eg errgroup.Group
	for _, file := range files {
		f := file // 创建局部变量避免闭包问题
		eg.Go(func() error {
			pluginName := strings.TrimSuffix(filepath.Base(f), ".so")
			m.config.mu.Lock()
			m.config.enabled[pluginName] = true
			m.config.mu.Unlock()

			return m.LoadPluginWithData(f)
		})
	}

	if err = eg.Wait(); err != nil {
		return err
	}

	return nil
}

// PublishPlugin 发布插件到插件市场
//
//	参数:
//	- info: 插件信息
//	返回:
//	- error: 发布过程中的错误
//	功能:
//	- 将插件信息添加到插件市场
//	- 使插件对其他用户可见
func (m *Manager) PublishPlugin(info PluginInfo) error {
	m.pluginMarket.AddPlugin(info)
	return nil
}

// InstallPlugin 下载并安装插件
//
//	参数:
//	- name: 插件名称
//	- version: 插件版本
//	返回:
//	- error: 下载或安装过程中的错误
//	功能:
//	- 从插件市场下载指定版本的插件
//	- 安装并初始化插件
//	- 更新版本信息
func (m *Manager) InstallPlugin(name, version string) error {
	// 这里我们假设插件已经在本地
	pluginPath := filepath.Join(m.pluginDir, fmt.Sprintf("%s_v%s.so", name, version))

	if err := m.LoadPlugin(pluginPath); err != nil {
		return err
	}

	m.versionManager.AddVersion(name, version)
	m.versionManager.SetActiveVersion(name, version)

	return nil
}

// ListAvailablePlugins 列出可用插件
//
//	返回:
//	- []PluginInfo: 可用插件信息列表
//	功能:
//	- 获取插件市场中所有可用插件的信息
//	- 包含插件名称、版本等元数据
func (m *Manager) ListAvailablePlugins() []PluginInfo {
	return m.pluginMarket.ListPlugins()
}

// HotUpdatePlugin 热更新插件
//
//	参数:
//	- name: 插件名称
//	- newVersion: 新版本号
//	返回:
//	- error: 更新过程中的错误
//	功能:
//	- 在不停止系统的情况下更新插件
//	- 保持原有配置
//	- 更新版本信息
func (m *Manager) HotUpdatePlugin(name, newVersion string) error {
	_, exists := m.versionManager.GetActiveVersion(name)
	if !exists {
		return newErrorf("插件未激活")
	}

	newPath := filepath.Join(m.pluginDir, fmt.Sprintf("%s_v%s.so", name, newVersion))
	if err := m.HotReload(name, newPath); err != nil {
		return err
	}

	m.versionManager.SetActiveVersion(name, newVersion)
	return nil
}

// RollbackPlugin 回滚插件版本
//
//	参数:
//	- name: 插件名称
//	- version: 目标版本号
//	返回:
//	- error: 回滚过程中的错误
//	功能:
//	- 将插件回滚到指定版本
//	- 恢复该版本的配置
//	- 更新版本信息
func (m *Manager) RollbackPlugin(name, version string) error {
	currentVersion, exists := m.versionManager.GetActiveVersion(name)
	if !exists {
		return newErrorf("插件未激活")
	}

	if currentVersion == version {
		return nil
	}

	targetPath := filepath.Join(m.pluginDir, fmt.Sprintf("%s_v%s.so", name, version))
	if err := m.HotReload(name, targetPath); err != nil {
		return err
	}

	m.versionManager.SetActiveVersion(name, version)
	return nil
}

// HasPermission 检查插件权限
//
//	EventName: 插件名称
//	action: 操作名称
//	功能:
//	- 验证插件是否有权限执行指定操作
func (m *Manager) HasPermission(pluginName, action string) bool {
	// 获取插件权限配置
	if perm, exists := m.permissions.Load(pluginName); exists {
		permission := perm.(PluginPermission)
		return permission.AllowedActions[action]
	}
	// 默认不允许未配置的操作
	return false
}

// SetPluginPermission 设置插件权限
//
//	EventName: 插件名称
//	permission: 权限配置
//	功能:
//	- 更新插件的权限配置
func (m *Manager) SetPluginPermission(pluginName string, permission *PluginPermission) {
	m.permissions.Store(pluginName, permission)
}

// RemovePluginPermission 移除插件权限
//
//	参数:
//	- EventName: 插件名称
//	功能:
//	- 从权限管理器中删除指定插件的所有权限配置
//	- 用于插件卸载或权限重置场景
func (m *Manager) RemovePluginPermission(pluginName string) {
	m.permissions.Delete(pluginName)
}

// LoadPluginPermissions 从配置加载插件权限
//
//	参数:
//	- permissions: 插件权限配置映射，key为插件名称，value为权限配置
//	功能:
//	- 批量导入插件权限配置
//	- 用于系统启动时初始化权限
//	- 支持动态更新多个插件的权限
func (m *Manager) LoadPluginPermissions(permissions map[string]*PluginPermission) {
	for name, perm := range permissions {
		m.permissions.Store(name, perm)
	}
}

// SetSandbox 设置沙箱环境
//
//	功能:
//	- 设置插件管理器的沙箱环境
func (m *Manager) SetSandbox(sandbox Sandbox) {
	m.sandbox = sandbox
}

// GetEventBus 获取事件总线
//
//	功能:
//	- 获取事件总线实例
func (m *Manager) GetEventBus() *eventBus {
	return m.eventBus
}

// GetVersionManager 获取版本管理器
//
//	功能:
//	- 获取版本管理器实例
func (m *Manager) GetVersionManager() *VersionManager {
	return m.versionManager
}

// GetPluginMarket 获取插件市场
//
//	功能:
//	- 获取插件市场实例
func (m *Manager) GetPluginMarket() *PluginMarket {
	return m.pluginMarket
}

func (m *Manager) preloadPlugin(name string) error {
	path := filepath.Join(m.pluginDir, name+".so")
	plugin := &lazyPlugin{path: path}

	// 预加载但不初始化
	if err := plugin.load(); err != nil {
		return err
	}

	m.preloadedPlugins.Store(name, plugin)
	return nil
}

// Serializer 序列化数据
//
//	参数:
//	- data: 需要序列化的数据
//	返回:
//	- []byte: 序列化后的字节数组
//	- error: 序列化过程中的错误
//	功能:
//	- 将任意类型数据序列化为字节数组
//	- 用于配置存储和数据传输
func Serializer(data any) ([]byte, error) {
	return msgpack.Marshal(data)
}

// Deserializer 反序列化数据
//
//	参数:
//	- data: 待反序列化的字节数组
//	- v: 目标结构体指针
//	返回:
//	- error: 反序列化过程中的错误
//	功能:
//	- 将字节数组反序列化为指定类型
//	- 用于配置加载和数据恢复
func Deserializer(data []byte, v any) error {
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
