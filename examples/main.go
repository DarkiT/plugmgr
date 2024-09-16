package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	pm "github.com/darkit/plugins"
)

type MyPluginConfig struct {
	Key string
}

type MyPluginInput struct {
	Data string
}

type MyPluginOutput struct {
	Result string
}

func main() {
	// 创建插件管理器
	manager, err := pm.NewManager("./plugins", "config.gob")
	if err != nil {
		log.Fatalf("创建插件管理器失败: %v", err)
	}

	// 订阅插件事件
	manager.SubscribeToEvent("PluginLoaded", func(event pm.Event) {
		fmt.Printf("插件加载事件: %+v\n", event)
	})

	// 加载插件
	pluginPath := filepath.Join("./plugins", "myplugin.so")
	initialConfig := MyPluginConfig{Key: "value"}
	if err := manager.LoadPluginWithData(pluginPath, initialConfig); err != nil {
		log.Fatalf("加载插件失败: %v", err)
	}

	// 获取已加载的插件列表
	loadedPlugins := manager.ListPlugins()
	fmt.Println("已加载的插件:", loadedPlugins)

	// 执行插件
	result, err := manager.ExecutePlugin("myplugin", "some data")
	if err != nil {
		log.Printf("执行插件失败: %v", err)
	} else {
		fmt.Printf("执行结果: %v\n", result)
	}

	// 执行插件 (泛型版本)
	input := MyPluginInput{Data: "some typed data"}
	typedResult, err := pm.ExecutePluginGeneric[MyPluginInput, MyPluginOutput](manager, "myplugin", input)
	if err != nil {
		log.Printf("执行插件失败 (泛型): %v", err)
	} else {
		fmt.Printf("执行结果 (泛型): %+v\n", typedResult)
	}

	// 获取插件统计信息
	stats, err := manager.GetPluginStats("myplugin")
	if err != nil {
		log.Printf("获取插件统计信息失败: %v", err)
	} else {
		fmt.Printf("插件统计信息: %+v\n", stats)
	}

	// 更新插件配置
	newConfig := map[string]interface{}{
		"new_key": "new_value",
	}
	if _, err := manager.ManagePluginConfig("myplugin", newConfig); err != nil {
		log.Printf("更新插件配置失败: %v", err)
	}

	// 禁用插件
	if err := manager.DisablePlugin("myplugin"); err != nil {
		log.Printf("禁用插件失败: %v", err)
	}

	// 启用插件
	if err := manager.EnablePlugin("myplugin"); err != nil {
		log.Printf("启用插件失败: %v", err)
	}

	// 热重载插件
	if err := manager.HotReload("myplugin", pluginPath); err != nil {
		log.Printf("热重载插件失败: %v", err)
	}

	// 卸载插件
	if err := manager.UnloadPlugin("myplugin"); err != nil {
		log.Printf("卸载插件失败: %v", err)
	}

	// 加载所有启用的插件
	if err := manager.LoadEnabledPlugins("./plugins"); err != nil {
		log.Printf("加载启用的插件失败: %v", err)
	}

	// 等待一段时间以便观察输出
	time.Sleep(2 * time.Second)
}
