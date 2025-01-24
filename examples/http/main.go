package main

import (
	"fmt"
	"log"
	"net/http"

	pm "github.com/darkit/plugmgr"
	"github.com/darkit/plugmgr/adapter"
)

func main() {
	// 创建插件管理器
	manager, err := pm.NewManager(
		"./plugins",        // 插件目录
		"./config.yaml",    // 配置文件路径
		"./public_key.pem", // 可选的公钥路径
	)
	if err != nil {
		log.Fatalf("初始化插件管理器失败: %v", err)
	}

	// 创建 HTTP 处理器
	Http := adapter.NewPluginHandler(manager, func(h http.HandlerFunc) http.HandlerFunc {
		return h
	})

	// 获取处理器接口
	HttpHandlers := Http.GetHandlers()

	// 创建路由
	mux := http.NewServeMux()

	// 插件管理路由
	mux.HandleFunc("/plugins", HttpHandlers.ListPlugins())
	mux.HandleFunc("/plugins/load/", HttpHandlers.LoadPlugin())
	mux.HandleFunc("/plugins/unload/", HttpHandlers.UnloadPlugin())
	mux.HandleFunc("/plugins/enable/", HttpHandlers.EnablePlugin())
	mux.HandleFunc("/plugins/disable/", HttpHandlers.DisablePlugin())
	mux.HandleFunc("/plugins/preload/", HttpHandlers.PreloadPlugin())
	mux.HandleFunc("/plugins/hotreload/", HttpHandlers.HotReloadPlugin())

	// 插件配置路由
	mux.HandleFunc("/plugins/config/", HttpHandlers.GetPluginConfig())
	mux.HandleFunc("/plugins/config/update/", HttpHandlers.UpdatedPluginConfig())

	// 插件权限路由
	mux.HandleFunc("/plugins/permission/", HttpHandlers.GetPluginPermission())
	mux.HandleFunc("/plugins/permission/set/", HttpHandlers.SetPluginPermission())
	mux.HandleFunc("/plugins/permission/remove/", HttpHandlers.RemovePluginPermission())

	// 插件统计路由
	mux.HandleFunc("/plugins/stats/", HttpHandlers.GetPluginStats())

	// 插件市场路由
	mux.HandleFunc("/market", HttpHandlers.ListMarketPlugins())
	mux.HandleFunc("/market/install/", HttpHandlers.InstallPlugin())
	mux.HandleFunc("/market/rollback/", HttpHandlers.RollbackPlugin())

	// 启动 HTTP 服务器
	fmt.Println("HTTP Server is running on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", mux))

	/*
		// 创建 Gin 处理器
		Gin := adapter.NewPluginHandler(manager, func(h http.HandlerFunc) gin.HandlerFunc {
			return func(c *gin.Context) {
				h(c.Writer, c.Request)
			}
		})

		// 获取处理器接口
		GinHandlers := Gin.GetHandlers()

		// 创建 Gin 路由
		r := gin.Default()

		// 插件管理路由
		r.GET("/plugins", GinHandlers.ListPlugins())
		r.POST("/plugins/load/:name", GinHandlers.LoadPlugin())
		r.POST("/plugins/unload/:name", GinHandlers.UnloadPlugin())
		r.POST("/plugins/enable/:name", GinHandlers.EnablePlugin())
		r.POST("/plugins/disable/:name", GinHandlers.DisablePlugin())
		r.POST("/plugins/preload/:name", GinHandlers.PreloadPlugin())
		r.POST("/plugins/hotreload/:name", GinHandlers.HotReloadPlugin())

		// 插件配置路由
		r.GET("/plugins/config/:name", GinHandlers.GetPluginConfig())
		r.PUT("/plugins/config/:name", GinHandlers.UpdatePluginConfig())

		// 插件权限路由
		r.GET("/plugins/permission/:name", GinHandlers.GetPluginPermission())
		r.PUT("/plugins/permission/:name", GinHandlers.SetPluginPermission())
		r.DELETE("/plugins/permission/:name", GinHandlers.RemovePluginPermission())

		// 插件统计路由
		r.GET("/plugins/stats/:name", GinHandlers.GetPluginStats())

		// 插件市场路由
		r.GET("/market", GinHandlers.ListMarketPlugins())
		r.POST("/market/install/:name", GinHandlers.InstallPlugin())
		r.POST("/market/rollback/:name", GinHandlers.RollbackPlugin())

		// 启动 Gin 服务器
		fmt.Println("Gin Server is running on http://localhost:8080")
		log.Fatal(r.Run(":8080"))
	*/
}
