package main

import (
	"fmt"
	"html/template"
	"net/http"

	pluginmanager "github.com/darkit/plugins"
	"github.com/darkit/plugins/manager/dist"
	"github.com/gin-gonic/gin"
)

type GinServer struct {
	manager *pluginmanager.Manager
}

func NewGinServer(manager *pluginmanager.Manager) *GinServer {
	return &GinServer{manager: manager}
}

func (s *GinServer) Start() error {
	r := gin.Default()

	// 使用嵌入的资源
	templ := template.Must(template.New("").ParseFS(dist.WebDist, "gin.html"))
	r.SetHTMLTemplate(templ)

	// 为嵌入的 Layui 文件提供服务
	r.StaticFS("/layui", http.FS(dist.WebDist))

	// 主页
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "gin.html", nil)
	})

	// API 路由
	api := r.Group("/api")
	{
		api.GET("/plugins", s.listPlugins)
		api.GET("/plugins/:name", s.getPlugin)
		api.POST("/plugins/:name/load", s.loadPlugin)
		api.POST("/plugins/:name/unload", s.unloadPlugin)
		api.POST("/plugins/:name/execute", s.executePlugin)
		api.POST("/plugins/:name/update", s.updatePlugin)
		api.POST("/plugins/:name/rollback", s.rollbackPlugin)
		api.GET("/market", s.listMarketPlugins)
		api.POST("/market/:name/install", s.installPlugin)
	}

	return r.Run(":8080")
}

func (s *GinServer) listPlugins(c *gin.Context) {
	plugins := s.manager.ListPlugins()
	c.JSON(http.StatusOK, plugins)
}

func (s *GinServer) getPlugin(c *gin.Context) {
	name := c.Param("name")
	plugin, err := s.manager.GetPluginConfig(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (s *GinServer) loadPlugin(c *gin.Context) {
	name := c.Param("name")
	err := s.manager.LoadPlugin(fmt.Sprintf("./plugins/%s.so", name))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (s *GinServer) unloadPlugin(c *gin.Context) {
	name := c.Param("name")
	err := s.manager.UnloadPlugin(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (s *GinServer) executePlugin(c *gin.Context) {
	name := c.Param("name")
	var data interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := s.manager.ExecutePlugin(name, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *GinServer) updatePlugin(c *gin.Context) {
	name := c.Param("name")
	var version string
	if err := c.ShouldBindJSON(&version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := s.manager.HotUpdatePlugin(name, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (s *GinServer) rollbackPlugin(c *gin.Context) {
	name := c.Param("name")
	var version string
	if err := c.ShouldBindJSON(&version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := s.manager.RollbackPlugin(name, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (s *GinServer) listMarketPlugins(c *gin.Context) {
	plugins := s.manager.ListAvailablePlugins()
	c.JSON(http.StatusOK, plugins)
}

func (s *GinServer) installPlugin(c *gin.Context) {
	name := c.Param("name")
	var version string
	if err := c.ShouldBindJSON(&version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := s.manager.DownloadAndInstallPlugin(name, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

//func main() {
//	manager, err := pluginmanager.NewManager("./plugins", "config.msgpack")
//	if err != nil {
//		panic(err)
//	}
//
//	server := NewGinServer(manager)
//	err = server.Start()
//	if err != nil {
//		panic(err)
//	}
//}
