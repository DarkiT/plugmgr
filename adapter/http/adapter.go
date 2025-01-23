package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/darkit/plugmgr/adapter"

	"github.com/darkit/plugmgr"
)

// HTTPHandler 标准http包的处理器实现
type HTTPHandler struct {
	*adapter.PluginHandler[http.HandlerFunc]
}

// HTTPContext http上下文适配器
type HTTPContext struct {
	w    http.ResponseWriter
	r    *http.Request
	vars map[string]string
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(manager *plugmgr.Manager) *HTTPHandler {
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := parsePathVars(r.URL.Path, getPatternForPath(r.URL.Path))
			// 将 HTTPContext 注入到请求上下文中
			ctx := context.WithValue(r.Context(), "pluginContext", &HTTPContext{w: w, r: r, vars: vars})
			r = r.WithContext(ctx)
			h(w, r)
		}
	}
	return &HTTPHandler{
		PluginHandler: adapter.NewPluginHandler(wrap, manager),
	}
}

// HandlePlugins 处理插件列表请求
func (h *HTTPHandler) HandlePlugins(ctx adapter.Context) {
	httpCtx := ctx.(*HTTPContext)
	switch httpCtx.Request().Method {
	case http.MethodGet:
		h.GetPlugins()(httpCtx.Response(), httpCtx.Request())
	default:
		httpCtx.Error(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandlePluginOperations 处理插件操作请求
func (h *HTTPHandler) HandlePluginOperations(ctx adapter.Context) {
	httpCtx := ctx.(*HTTPContext)
	// name := httpCtx.Param("name")
	action := httpCtx.Param("action")

	switch action {
	case "load":
		h.LoadPlugin()(httpCtx.Response(), httpCtx.Request())
	case "unload":
		h.UnloadPlugin()(httpCtx.Response(), httpCtx.Request())
	case "execute":
		h.ExecutePlugin()(httpCtx.Response(), httpCtx.Request())
	case "config":
		switch httpCtx.Request().Method {
		case http.MethodGet:
			h.GetPluginConfig()(httpCtx.Response(), httpCtx.Request())
		case http.MethodPut:
			h.UpdatePluginConfig()(httpCtx.Response(), httpCtx.Request())
		}
	case "enable":
		h.EnablePlugin()(httpCtx.Response(), httpCtx.Request())
	case "disable":
		h.DisablePlugin()(httpCtx.Response(), httpCtx.Request())
	default:
		httpCtx.Error(http.StatusNotFound, "Operation not found")
	}
}

// HandleMarket 处理插件市场请求
func (h *HTTPHandler) HandleMarket(ctx adapter.Context) {
	httpCtx := ctx.(*HTTPContext)
	switch httpCtx.Request().Method {
	case http.MethodGet:
		h.GetMarketPlugins()(httpCtx.Response(), httpCtx.Request())
	default:
		httpCtx.Error(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleMarketOperations 处理插件市场操作
func (h *HTTPHandler) HandleMarketOperations(ctx adapter.Context) {
	httpCtx := ctx.(*HTTPContext)
	// name := httpCtx.Param("name")
	action := httpCtx.Param("action")

	switch action {
	case "install":
		h.InstallPlugin()(httpCtx.Response(), httpCtx.Request())
	case "update":
		h.UpdatePlugin()(httpCtx.Response(), httpCtx.Request())
	case "rollback":
		h.RollbackPlugin()(httpCtx.Response(), httpCtx.Request())
	default:
		httpCtx.Error(http.StatusNotFound, "Operation not found")
	}
}

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/plugins", func(w http.ResponseWriter, r *http.Request) {
		h.HandlePlugins(&HTTPContext{w: w, r: r})
	})

	mux.HandleFunc("/plugins/", func(w http.ResponseWriter, r *http.Request) {
		vars := parsePathVars(r.URL.Path, "/plugins/{name}/{action}")
		h.HandlePluginOperations(&HTTPContext{w: w, r: r, vars: vars})
	})

	mux.HandleFunc("/market", func(w http.ResponseWriter, r *http.Request) {
		h.HandleMarket(&HTTPContext{w: w, r: r})
	})

	mux.HandleFunc("/market/", func(w http.ResponseWriter, r *http.Request) {
		vars := parsePathVars(r.URL.Path, "/market/{name}/{action}")
		h.HandleMarketOperations(&HTTPContext{w: w, r: r, vars: vars})
	})

	return mux
}

func (c *HTTPContext) Request() *http.Request        { return c.r }
func (c *HTTPContext) Response() http.ResponseWriter { return c.w }
func (c *HTTPContext) Param(name string) string {
	return c.vars[name]
}

func (c *HTTPContext) JSON(code int, obj interface{}) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(code)
	return json.NewEncoder(c.w).Encode(obj)
}

func (c *HTTPContext) Error(code int, msg string) error {
	return c.JSON(code, map[string]string{"error": msg})
}

func (c *HTTPContext) Bind(obj interface{}) error {
	return json.NewDecoder(c.r.Body).Decode(obj)
}

// 辅助函数：解析路径参数
func parsePathVars(path, pattern string) map[string]string {
	vars := make(map[string]string)
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")

	if len(pathParts) != len(patternParts) {
		return vars
	}

	for i, pattern := range patternParts {
		if strings.HasPrefix(pattern, "{") && strings.HasSuffix(pattern, "}") {
			key := strings.Trim(pattern, "{}")
			vars[key] = pathParts[i]
		}
	}

	return vars
}

// 辅助函数：根据路径获取对应的模式
func getPatternForPath(path string) string {
	if strings.HasPrefix(path, "/plugins/") {
		return "/plugins/{name}/{action}"
	}
	if strings.HasPrefix(path, "/market/") {
		return "/market/{name}/{action}"
	}
	return ""
}
