package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/darkit/plugmgr"
)

// WebAdapter 定义了插件系统与Web框架之间的适配器接口
type WebAdapter interface {
	// HandlePlugins 处理插件列表请求
	HandlePlugins(ctx Context)

	// HandlePluginOperations 处理插件操作请求
	HandlePluginOperations(ctx Context)

	// HandleMarket 处理插件市场请求
	HandleMarket(ctx Context)

	// HandleMarketOperations 处理插件市场操作
	HandleMarketOperations(ctx Context)
}

// Context 定义了通用的上下文接口
type Context interface {
	// Request 获取请求信息
	Request() *http.Request
	// Response 获取响应写入器
	Response() http.ResponseWriter
	// Param 获取路由参数
	Param(name string) string
	// JSON 返回JSON响应
	JSON(code int, obj interface{}) error
	// Error 返回错误响应
	Error(code int, msg string) error
	// Bind 绑定请求数据
	Bind(obj interface{}) error
}

// Handler 定义了所有插件处理方法的泛型接口
type Handler[T any] interface {
	// 插件基础操作
	GetPlugins() T
	LoadPlugin() T
	UnloadPlugin() T
	ExecutePlugin() T

	// 插件配置操作
	GetPluginConfig() T
	UpdatePluginConfig() T

	// 插件市场操作
	GetMarketPlugins() T
	InstallPlugin() T

	// 插件版本操作
	UpdatePlugin() T
	RollbackPlugin() T
}

// PluginHandler 泛型结构体，实现 Handler 接口
type PluginHandler[T any] struct {
	ctx     context.Context
	wrap    func(http.HandlerFunc) T
	manager *plugmgr.Manager
	logger  *slog.Logger
}

// NewPluginHandler 创建新的处理器实例
func NewPluginHandler[T any](wrap func(http.HandlerFunc) T, manager *plugmgr.Manager) *PluginHandler[T] {
	return &PluginHandler[T]{
		ctx:     context.Background(),
		wrap:    wrap,
		manager: manager,
		logger:  slog.Default(),
	}
}

// GetHandlers 返回实现了 Handler 接口的 PluginHandler
func (h *PluginHandler[T]) GetHandlers() Handler[T] {
	return h
}

// GetPlugins 获取插件列表
func (h *PluginHandler[T]) GetPlugins() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		plugins := h.manager.ListPlugins()
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code":  0,
			"msg":   "获取成功",
			"count": len(plugins),
			"data":  plugins,
		})
	})
}

// LoadPlugin 加载插件
func (h *PluginHandler[T]) LoadPlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")

		err := h.manager.LoadPlugin(fmt.Sprintf("./plugins/%s.so", name))
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "加载成功",
		})
	})
}

// ExecutePlugin 执行插件
func (h *PluginHandler[T]) ExecutePlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")

		var data struct {
			Action string      `json:"action"`
			Params interface{} `json:"params"`
		}

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			errorResponse(w, http.StatusBadRequest, "无效的请求数据")
			return
		}

		if !h.manager.HasPermission(name, data.Action) {
			errorResponse(w, http.StatusForbidden, "操作未授权")
			return
		}

		result, err := h.manager.ExecutePlugin(name, data.Params)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "执行成功",
			"data": result,
		})
	})
}

// UnloadPlugin 卸载插件
func (h *PluginHandler[T]) UnloadPlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")
		err := h.manager.UnloadPlugin(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "卸载成功",
		})
	})
}

// GetPluginConfig 获取插件配置
func (h *PluginHandler[T]) GetPluginConfig() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")
		config, err := h.manager.GetPluginConfig(name)
		if err != nil {
			errorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "获取成功",
			"data": config,
		})
	})
}

// UpdatePluginConfig 更新插件配置
func (h *PluginHandler[T]) UpdatePluginConfig() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")

		var config json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			errorResponse(w, http.StatusBadRequest, "无效的配置数据")
			return
		}

		updatedConfig, err := h.manager.ManagePluginConfig(name, config)
		if err != nil {
			switch {
			case errors.Is(err, plugmgr.ErrPluginNotFound):
				errorResponse(w, http.StatusNotFound, "插件未找到")
			default:
				errorResponse(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "更新成功",
			"data": updatedConfig,
		})
	})
}

// GetMarketPlugins 获取插件市场列表
func (h *PluginHandler[T]) GetMarketPlugins() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		plugins := h.manager.ListAvailablePlugins()
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code":  0,
			"msg":   "获取成功",
			"count": len(plugins),
			"data":  plugins,
		})
	})
}

// InstallPlugin 安装插件
func (h *PluginHandler[T]) InstallPlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")
		var version string
		if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
			errorResponse(w, http.StatusBadRequest, "无效的版本格式")
			return
		}

		err := h.manager.DownloadAndInstallPlugin(name, version)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "安装成功",
		})
	})
}

// UpdatePlugin 更新插件
func (h *PluginHandler[T]) UpdatePlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")
		var version string
		if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
			errorResponse(w, http.StatusBadRequest, "无效的版本格式")
			return
		}

		err := h.manager.HotUpdatePlugin(name, version)
		if err != nil {
			switch {
			case errors.Is(err, plugmgr.ErrPluginNotFound):
				errorResponse(w, http.StatusNotFound, "插件未找到")
			case errors.Is(err, plugmgr.ErrIncompatibleVersion):
				errorResponse(w, http.StatusBadRequest, "版本不兼容")
			default:
				errorResponse(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "更新成功",
		})
	})
}

// RollbackPlugin 回滚插件版本
func (h *PluginHandler[T]) RollbackPlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")

		var version string
		if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
			errorResponse(w, http.StatusBadRequest, "无效的版本格式")
			return
		}

		err := h.manager.RollbackPlugin(name, version)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "回滚成功",
		})
	})
}

// EnablePlugin 启用插件
func (h *PluginHandler[T]) EnablePlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")
		err := h.manager.EnablePlugin(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "启用成功",
		})
	})
}

// DisablePlugin 禁用插件
func (h *PluginHandler[T]) DisablePlugin() T {
	return h.wrap(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context().Value("pluginContext").(Context)
		name := ctx.Param("name")
		err := h.manager.DisablePlugin(name)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"code": 0,
			"msg":  "禁用成功",
		})
	})
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
