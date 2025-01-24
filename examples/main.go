package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	pm "github.com/darkit/plugmgr"
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

type FileManagerConfig struct {
	WorkDir           string
	AllowedExtensions []string
	MaxFileSize       int
}

type FileOperation struct {
	Operation string
	Filename  string
	Content   string
}

func main() {
	// 创建插件管理器
	manager, err := pm.NewManager("./plugins", "config.db")
	if err != nil {
		log.Fatalf("创建插件管理器失败: %v", err)
	}

	// 订阅插件事件
	manager.SubscribeToEvent("PluginLoaded", func(event pm.Event) {
		fmt.Printf("插件加载事件: %+v\n", event)
	})

	// 加载插件
	pluginPath := filepath.Join("./plugins", "myplugin.so")
	initialConfig := FileManagerConfig{
		WorkDir:           "./files",
		AllowedExtensions: []string{".txt", ".log", ".json"},
		MaxFileSize:       10,
	}
	if err := manager.LoadPluginWithData(pluginPath, initialConfig); err != nil {
		log.Fatalf("加载插件失败: %v", err)
	}

	// 演示文件操作
	fmt.Println("\n=== 开始文件操作演示 ===")

	// 1. 创建文件
	createOp := FileOperation{
		Operation: "create",
		Filename:  "test.txt",
		Content:   "Hello, World!",
	}
	result, err := manager.ExecutePlugin("myplugin", createOp)
	if err != nil {
		log.Printf("创建文件失败: %v", err)
	} else {
		fmt.Printf("创建文件结果: %+v\n", result)
	}

	// 2. 读取文件
	readOp := FileOperation{
		Operation: "read",
		Filename:  "test.txt",
	}
	result, err = manager.ExecutePlugin("myplugin", readOp)
	if err != nil {
		log.Printf("读取文件失败: %v", err)
	} else {
		fmt.Printf("读取文件结果: %+v\n", result)
	}

	// 3. 更新文件
	updateOp := FileOperation{
		Operation: "update",
		Filename:  "test.txt",
		Content:   "Updated content",
	}
	result, err = manager.ExecutePlugin("myplugin", updateOp)
	if err != nil {
		log.Printf("更新文件失败: %v", err)
	} else {
		fmt.Printf("更新文件结果: %+v\n", result)
	}

	// 4. 再次读取文件验证更新
	result, err = manager.ExecutePlugin("myplugin", readOp)
	if err != nil {
		log.Printf("验证更新失败: %v", err)
	} else {
		fmt.Printf("更新后的文件内容: %+v\n", result)
	}

	// 5. 删除文件
	deleteOp := FileOperation{
		Operation: "delete",
		Filename:  "test.txt",
	}
	result, err = manager.ExecutePlugin("myplugin", deleteOp)
	if err != nil {
		log.Printf("删除文件失败: %v", err)
	} else {
		fmt.Printf("删除文件结果: %+v\n", result)
	}

	fmt.Println("=== 文件操作演示结束 ===")

	// 获取已加载的插件列表
	loadedPlugins := manager.ListPlugins()
	fmt.Println("已加载的插件:", loadedPlugins)

	// 执行插件
	result, err = manager.ExecutePlugin("myplugin", "some data")
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
	if _, err = manager.ConfigUpdated("myplugin", newConfig); err != nil {
		log.Printf("更新插件配置失败: %v", err)
	}

	// 禁用插件
	if err = manager.DisablePlugin("myplugin"); err != nil {
		log.Printf("禁用插件失败: %v", err)
	}

	// 启用插件
	if err = manager.EnablePlugin("myplugin"); err != nil {
		log.Printf("启用插件失败: %v", err)
	}

	// 热重载插件
	if err = manager.HotReload("myplugin", pluginPath); err != nil {
		log.Printf("热重载插件失败: %v", err)
	}

	// 卸载插件
	if err = manager.UnloadPlugin("myplugin"); err != nil {
		log.Printf("卸载插件失败: %v", err)
	}

	// 加载所有启用的插件
	if err = manager.LoadEnabledPlugins("./plugins"); err != nil {
		log.Printf("加载启用的插件失败: %v", err)
	}

	// 等待一段时间以便观察输出
	time.Sleep(2 * time.Second)
}
