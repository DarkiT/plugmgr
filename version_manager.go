package pluginmanager

import (
	"sort"
	"sync"
)

type VersionManager struct {
	versions      map[string][]string // 插件名称 -> 版本列表
	activeVersion map[string]string   // 插件名称 -> 当前激活版本
	mu            sync.RWMutex
}

func NewVersionManager() *VersionManager {
	return &VersionManager{
		versions:      make(map[string][]string),
		activeVersion: make(map[string]string),
	}
}

func (vm *VersionManager) AddVersion(pluginName, version string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, exists := vm.versions[pluginName]; !exists {
		vm.versions[pluginName] = []string{}
	}
	vm.versions[pluginName] = append(vm.versions[pluginName], version)
	sort.Slice(vm.versions[pluginName], func(i, j int) bool {
		return compareVersions(vm.versions[pluginName][i], vm.versions[pluginName][j]) > 0
	})
}

func (vm *VersionManager) SetActiveVersion(pluginName, version string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vm.activeVersion[pluginName] = version
}

func (vm *VersionManager) GetActiveVersion(pluginName string) (string, bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	version, exists := vm.activeVersion[pluginName]
	return version, exists
}

func (vm *VersionManager) GetVersions(pluginName string) []string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return vm.versions[pluginName]
}

// 插件市场相关代码

type PluginInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Versions    []string `json:"versions"`
}

type PluginMarket struct {
	plugins map[string]PluginInfo
	mu      sync.RWMutex
}

func NewPluginMarket() *PluginMarket {
	return &PluginMarket{
		plugins: make(map[string]PluginInfo),
	}
}

func (pm *PluginMarket) AddPlugin(info PluginInfo) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if existing, exists := pm.plugins[info.Name]; exists {
		existing.Versions = append(existing.Versions, info.Version)
		sort.Slice(existing.Versions, func(i, j int) bool {
			return compareVersions(existing.Versions[i], existing.Versions[j]) > 0
		})
		pm.plugins[info.Name] = existing
	} else {
		info.Versions = []string{info.Version}
		pm.plugins[info.Name] = info
	}
}

func (pm *PluginMarket) GetPlugin(name string) (PluginInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	info, exists := pm.plugins[name]
	return info, exists
}

func (pm *PluginMarket) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var result []PluginInfo
	for _, info := range pm.plugins {
		result = append(result, info)
	}
	return result
}
