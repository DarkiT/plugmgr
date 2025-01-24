package adapter

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/darkit/plugmgr"
)

// Handler 是一个泛型接口，定义了所有处理方法
type Handler[T any] interface {
	// 基础插件管理
	ListPlugins() T
	LoadPlugin() T
	UnloadPlugin() T
	EnablePlugin() T
	DisablePlugin() T
	PreloadPlugin() T
	HotReloadPlugin() T

	// 插件配置
	GetPluginConfig() T
	UpdatedPluginConfig() T

	// 插件执行
	ExecutePlugin() T

	// 插件权限
	GetPluginPermission() T
	SetPluginPermission() T
	RemovePluginPermission() T

	// 插件市场
	ListMarketPlugins() T
	InstallPlugin() T
	RollbackPlugin() T

	// 插件统计
	GetPluginStats() T
}

// PluginHandler 是一个泛型结构体，实现了 Handler 接口
type PluginHandler[T any] struct {
	manager *plugmgr.Manager
	warp    func(http.HandlerFunc) T
}

// NewPluginHandler 创建一个新的 PluginHandler 实例
func NewPluginHandler[T any](manager *plugmgr.Manager, warp func(http.HandlerFunc) T) *PluginHandler[T] {
	return &PluginHandler[T]{
		manager: manager,
		warp:    warp,
	}
}

// GetHandlers 返回实现了 Handler 接口的 PluginHandler
func (h *PluginHandler[T]) GetHandlers() Handler[T] {
	return h
}

// ListPlugins 获取插件列表
func (h *PluginHandler[T]) ListPlugins() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		plugins := h.manager.ListPlugins()
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": plugins,
		})
	})
}

// LoadPlugin 加载插件
func (h *PluginHandler[T]) LoadPlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		err := h.manager.LoadPlugin(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件加载成功",
		})
	})
}

// UnloadPlugin 卸载插件
func (h *PluginHandler[T]) UnloadPlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		err := h.manager.UnloadPlugin(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件卸载成功",
		})
	})
}

// EnablePlugin 启用插件
func (h *PluginHandler[T]) EnablePlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		err := h.manager.EnablePlugin(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件启用成功",
		})
	})
}

// DisablePlugin 禁用插件
func (h *PluginHandler[T]) DisablePlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		err := h.manager.DisablePlugin(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件禁用成功",
		})
	})
}

// GetPluginConfig 获取插件配置
func (h *PluginHandler[T]) GetPluginConfig() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		config, err := h.manager.GetPluginConfig(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": config,
		})
	})
}

// UpdatedPluginConfig 更新插件配置
func (h *PluginHandler[T]) UpdatedPluginConfig() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "参数格式错误")
			return
		}
		conf, err := h.manager.ConfigUpdated(name, body)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "配置更新成功",
			"data": conf,
		})
	})
}

// ExecutePlugin 执行插件
func (h *PluginHandler[T]) ExecutePlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "参数格式错误")
			return
		}
		result, err := h.manager.ExecutePlugin(name, body)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": result,
		})
	})
}

// ListMarketPlugins 获取市场插件列表
func (h *PluginHandler[T]) ListMarketPlugins() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": h.manager.ListAvailablePlugins(),
		})
	})
}

// InstallPlugin 安装插件
func (h *PluginHandler[T]) InstallPlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		version := r.URL.Query().Get("version")
		err := h.manager.InstallPlugin(name, version)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件安装成功",
		})
	})
}

// RollbackPlugin 回滚插件
func (h *PluginHandler[T]) RollbackPlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		version := r.URL.Query().Get("version")
		err := h.manager.RollbackPlugin(name, version)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件回滚成功",
		})
	})
}

// PreloadPlugin 预加载插件
func (h *PluginHandler[T]) PreloadPlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		err := h.manager.PreloadPlugins([]string{name})
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件预加载成功",
		})
	})
}

// HotReloadPlugin 热重载插件
func (h *PluginHandler[T]) HotReloadPlugin() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		var params struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			errorResponse(w, http.StatusBadRequest, "参数格式错误")
			return
		}
		err := h.manager.HotReload(name, params.Path)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "插件热重载成功",
		})
	})
}

// GetPluginPermission 获取插件权限
func (h *PluginHandler[T]) GetPluginPermission() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		hasPermission := h.manager.HasPermission(name, "read")
		if !hasPermission {
			errorResponse(w, http.StatusForbidden, "无权限访问")
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": hasPermission,
		})
	})
}

// SetPluginPermission 设置插件权限
func (h *PluginHandler[T]) SetPluginPermission() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		var permission plugmgr.PluginPermission
		if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
			errorResponse(w, http.StatusBadRequest, "权限格式错误")
			return
		}
		h.manager.SetPluginPermission(name, &permission)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "权限设置成功",
		})
	})
}

// RemovePluginPermission 移除插件权限
func (h *PluginHandler[T]) RemovePluginPermission() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		h.manager.RemovePluginPermission(name)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "权限移除成功",
		})
	})
}

// GetPluginStats 获取插件统计信息
func (h *PluginHandler[T]) GetPluginStats() T {
	return h.warp(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		stats, err := h.manager.GetPluginStats(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "获取统计信息失败")
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": stats,
		})
	})
}

// SetupRoutes 设置路由
func (h *PluginHandler[T]) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	setupRoute := func(path string, handler func() T) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			h := handler()
			if fn, ok := any(h).(func(http.ResponseWriter, *http.Request)); ok {
				fn(w, r)
			} else {
				errorResponse(w, http.StatusInternalServerError, "Handler type mismatch")
			}
		})
	}

	// 插件管理路由
	setupRoute("/plugins", h.ListPlugins)
	setupRoute("/plugins/load/", h.LoadPlugin)
	setupRoute("/plugins/unload/", h.UnloadPlugin)
	setupRoute("/plugins/enable/", h.EnablePlugin)
	setupRoute("/plugins/disable/", h.DisablePlugin)
	setupRoute("/plugins/preload/", h.PreloadPlugin)
	setupRoute("/plugins/hotreload/", h.HotReloadPlugin)

	// 插件配置路由
	setupRoute("/plugins/config/", h.GetPluginConfig)
	setupRoute("/plugins/config/update/", h.UpdatedPluginConfig)

	// 插件权限路由
	setupRoute("/plugins/permission/", h.GetPluginPermission)
	setupRoute("/plugins/permission/set/", h.SetPluginPermission)
	setupRoute("/plugins/permission/remove/", h.RemovePluginPermission)

	// 插件统计路由
	setupRoute("/plugins/stats/", h.GetPluginStats)

	// 插件市场路由
	setupRoute("/market", h.ListMarketPlugins)
	setupRoute("/market/install/", h.InstallPlugin)
	setupRoute("/market/rollback/", h.RollbackPlugin)

	return mux
}

// 辅助函数
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]interface{}{
		"code": -1,
		"msg":  message,
	})
}
