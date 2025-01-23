package gin

import (
	"context"
	"net/http"

	"github.com/darkit/plugmgr"
	"github.com/darkit/plugmgr/adapter"
	"github.com/gin-gonic/gin"
)

// GinHandler Gin框架的处理器实现
type GinHandler struct {
	*adapter.PluginHandler[gin.HandlerFunc]
}

// GinContext Gin上下文适配器
type GinContext struct {
	ctx *gin.Context
}

// NewGinHandler 创建Gin处理器
func NewGinHandler(manager *plugmgr.Manager) *GinHandler {
	wrap := func(h http.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			// 将 GinContext 注入到请求上下文中
			ctx := context.WithValue(c.Request.Context(), "pluginContext", &GinContext{ctx: c})
			c.Request = c.Request.WithContext(ctx)
			h(c.Writer, c.Request)
		}
	}
	return &GinHandler{
		PluginHandler: adapter.NewPluginHandler(wrap, manager),
	}
}

// HandlePlugins 处理插件列表请求
func (h *GinHandler) HandlePlugins(ctx adapter.Context) {
	ginCtx := ctx.(*GinContext)
	switch ginCtx.Request().Method {
	case http.MethodGet:
		h.GetPlugins()(ginCtx.ctx)
	default:
		ginCtx.Error(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandlePluginOperations 处理插件操作请求
func (h *GinHandler) HandlePluginOperations(ctx adapter.Context) {
	ginCtx := ctx.(*GinContext)
	// name := ginCtx.Param("name")
	action := ginCtx.Param("action")

	switch action {
	case "load":
		h.LoadPlugin()(ginCtx.ctx)
	case "unload":
		h.UnloadPlugin()(ginCtx.ctx)
	case "execute":
		h.ExecutePlugin()(ginCtx.ctx)
	case "config":
		switch ginCtx.Request().Method {
		case http.MethodGet:
			h.GetPluginConfig()(ginCtx.ctx)
		case http.MethodPut:
			h.UpdatePluginConfig()(ginCtx.ctx)
		}
	case "enable":
		h.EnablePlugin()(ginCtx.ctx)
	case "disable":
		h.DisablePlugin()(ginCtx.ctx)
	default:
		ginCtx.Error(http.StatusNotFound, "Operation not found")
	}
}

// HandleMarket 处理插件市场请求
func (h *GinHandler) HandleMarket(ctx adapter.Context) {
	ginCtx := ctx.(*GinContext)
	switch ginCtx.Request().Method {
	case http.MethodGet:
		h.GetMarketPlugins()(ginCtx.ctx)
	default:
		ginCtx.Error(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleMarketOperations 处理插件市场操作
func (h *GinHandler) HandleMarketOperations(ctx adapter.Context) {
	ginCtx := ctx.(*GinContext)
	// name := ginCtx.Param("name")
	action := ginCtx.Param("action")

	switch action {
	case "install":
		h.InstallPlugin()(ginCtx.ctx)
	case "update":
		h.UpdatePlugin()(ginCtx.ctx)
	case "rollback":
		h.RollbackPlugin()(ginCtx.ctx)
	default:
		ginCtx.Error(http.StatusNotFound, "Operation not found")
	}
}

// RegisterRoutes 注册Gin路由
func (h *GinHandler) RegisterRoutes(r *gin.Engine) {
	r.Any("/plugins", func(c *gin.Context) {
		h.HandlePlugins(&GinContext{ctx: c})
	})
	r.Any("/plugins/:name/:action", func(c *gin.Context) {
		h.HandlePluginOperations(&GinContext{ctx: c})
	})
	r.Any("/market", func(c *gin.Context) {
		h.HandleMarket(&GinContext{ctx: c})
	})
	r.Any("/market/:name/:action", func(c *gin.Context) {
		h.HandleMarketOperations(&GinContext{ctx: c})
	})
}

func (c *GinContext) Request() *http.Request        { return c.ctx.Request }
func (c *GinContext) Response() http.ResponseWriter { return c.ctx.Writer }
func (c *GinContext) Param(name string) string      { return c.ctx.Param(name) }
func (c *GinContext) JSON(code int, obj interface{}) error {
	c.ctx.JSON(code, obj)
	return nil
}

func (c *GinContext) Error(code int, msg string) error {
	c.ctx.JSON(code, gin.H{
		"code":  -1,
		"error": msg,
	})
	return nil
}
func (c *GinContext) Bind(obj interface{}) error { return c.ctx.ShouldBindJSON(obj) }
