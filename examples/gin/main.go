package main

import (
	"log/slog"

	pm "github.com/darkit/plugmgr"
	adapter "github.com/darkit/plugmgr/adapter/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.Default()

	// 初始化插件管理器
	manager, err := pm.NewManager(
		"./plugins",        // 插件目录
		"./config.yaml",    // 配置文件路径
		"./public_key.pem", // 可选的公钥路径
	)
	if err != nil {
		logger.Error("初始化插件管理器失败", "error", err.Error())
		return
	}

	// 设置默认权限
	manager.LoadPluginPermissions(map[string]*pm.PluginPermission{
		"demo": {
			AllowedActions: map[string]bool{
				"execute": true,
				"config":  true,
			},
			Roles: []string{"admin"},
		},
	})

	// 创建 Gin 引擎
	r := gin.Default()

	// 创建并注册适配器
	handler := adapter.NewGinHandler(manager)
	handler.RegisterRoutes(r)

	// 启动服务器
	if err = r.Run(":8080"); err != nil {
		logger.Error("启动服务器失败", "error", err.Error())
	}
}
